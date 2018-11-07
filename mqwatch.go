package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/streadway/amqp"
)

// message contains info about a single message, as stored in the in-mem buffer
type message struct {
	Seq        int64
	Body       []byte
	RoutingKey string
	Received   time.Time
	Sender     string
	Headers    amqp.Table
}

// query contains the query text and the channel on which to send the query response
type query struct {
	rawQuery string
	spec     []querySpec
	respch   chan queryResult
}

// querySpec represents a structured query:
// words that need to be present in the message, fields that need to exist with specific values
// range of sequence number.
type querySpec struct {
	words      []string
	routingKey string
	seqFrom    int64
	seqTo      int64
}

// queryResult contains the messages matching a query and the current sequence number (i.e. the number of msgs received so far)
type queryResult struct {
	messages []message
	seq      int64
	bufSize  int
}

// config contains all configurable parameters of the application
type config struct {
	url         string
	exchanges   []string
	key         string
	port        int
	bufferSize  int
	maxResults  int
	prettyPrint bool
}

// receiverChannels bundles all channels that the receiver goroutine needs
type receiverChannels struct {
	reqs  chan query
	msgs  <-chan amqp.Delivery
	ctrl  chan string
	dumps chan chan []message
}

func parseSpec(input string) []querySpec {
	splt := regexp.MustCompile("\n|,")
	ws := regexp.MustCompile(" +")
	exp := regexp.MustCompile(`(\w+):(.*)|#(\d+)?-(\d+)?`)
	var result []querySpec
	for _, line := range splt.Split(input, -1) {
		spec := querySpec{seqTo: math.MaxInt64, routingKey: "#"}
		for _, tok := range ws.Split(line, -1) {
			sm := exp.FindStringSubmatch(tok)
			if sm != nil {
				if len(sm[1]) > 0 {
					switch sm[1] {
					case "key":
						spec.routingKey = sm[2]
					}
				} else {
					if len(sm[3]) > 0 {
						from, _ := strconv.Atoi(sm[3])
						spec.seqFrom = int64(from)
					}
					if len(sm[4]) > 0 {
						to, _ := strconv.Atoi(sm[4])
						spec.seqTo = int64(to)
					}
				}
			} else {
				spec.words = append(spec.words, tok)
			}
		}
		result = append(result, spec)
	}
	return result
}

func accept1(m message, spec querySpec) bool {
	if spec.seqFrom > m.Seq || spec.seqTo < m.Seq {
		return false
	}
	for _, w := range spec.words {
		if !bytes.Contains(m.Body, []byte(w)) {
			return false
		}
	}
	if spec.routingKey != "#" && spec.routingKey != m.RoutingKey {
		return false
	}
	return true
}

func accept(m message, specs []querySpec) bool {
	for _, spec := range specs {
		if accept1(m, spec) {
			return true
		}
	}
	return false
}

func receive(cs receiverChannels, cfg config) {
	reverse := func(buf []message) {
		for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
			buf[i], buf[j] = buf[j], buf[i]
		}
	}
	buf := []message{}
	seq := int64(0)
	for {
		select {
		case cmd := <-cs.ctrl:
			switch cmd {
			case "clear":
				l := len(buf)
				buf = nil
				log.Printf("Cleared message buffer (%d messages deleted)\n", l)
			default:
				log.Printf("Unknown control command %s\n", cmd)
			}
		case msg := <-cs.msgs:
			// Unfortunately, msg.Timestamp is empty, so we can't use it.
			buf = append(buf, message{seq, msg.Body, msg.RoutingKey, time.Now(), msg.AppId, msg.Headers})
			seq++
			l := len(buf)
			if l > cfg.bufferSize {
				buf = buf[l-cfg.bufferSize:]
			}
		case q := <-cs.reqs:
			log.Printf("Processing query: %s\n", q.rawQuery)
			var r []message
			if string(q.rawQuery) == "" {
				l := cfg.maxResults
				if len(buf) < l {
					l = len(buf)
				}
				r = make([]message, l)
				copy(r, buf[len(buf)-l:len(buf)])
			} else {
				for i := len(buf) - 1; i >= 0; i-- {
					if accept(buf[i], q.spec) {
						r = append(r, buf[i])
						if len(r) == cfg.maxResults {
							break
						}
					}
				}
				reverse(r)
			}
			q.respch <- queryResult{messages: r, seq: seq, bufSize: len(buf)}
		case dc := <-cs.dumps:
			bufCopy := make([]message, len(buf))
			copy(bufCopy, buf)
			dc <- bufCopy
		}
	}
}

func frequencies(ms []message) map[string]int {
	re := regexp.MustCompile("[a-f0-9]{8}(-[a-f0-9]{4}){3}-[a-f0-9]{12}")
	f := func(key string) string {
		return re.ReplaceAllString(key, "<uuid>")
	}
	freq := make(map[string]int)
	for _, m := range ms {
		key := f(m.RoutingKey)
		cnt, ok := freq[key]
		if ok {
			freq[key] = cnt + 1
		} else {
			freq[key] = 1
		}
	}
	return freq
}

func handleIndex(cfg config, querych chan<- query, reqStr string, w http.ResponseWriter) {
	var result queryResult
	respch := make(chan queryResult)
	querych <- query{reqStr, parseSpec(reqStr), respch}
	result = <-respch
	t := templateIndexHTML(cfg.prettyPrint)
	w.Header().Set("Content-Type", "text/html")
	err := t.Execute(w, indexHTMLContent{
		Created:       time.Now(),
		Frequencies:   frequencies(result.messages),
		Exchanges:     cfg.exchanges,
		Messages:      result.messages,
		Query:         reqStr,
		ReceivedTotal: result.seq,
		BufferSize:    result.bufSize})
	if err != nil {
		log.Printf("Shit happened: %v\n", err)
	}
}

func handleDump(dumpch chan<- chan []message, w http.ResponseWriter) {
	d := make(chan []message)
	dumpch <- d
	msgs := <-d
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[\n"))
	for i, m := range msgs {
		if i > 0 {
			w.Write([]byte(",\n"))
		}
		w.Write(m.Body)
	}
	w.Write([]byte("\n]\n"))
}

func queryHandler(cfg config, cs receiverChannels) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/clear" {
			cs.ctrl <- "clear"
			http.Redirect(w, r, "/", 301)
		} else if r.URL.Path == "/dump" {
			handleDump(cs.dumps, w)
		} else if r.URL.Path == "/" {
			handleIndex(cfg, cs.reqs, r.URL.Query().Get("q"), w)
		}
	}
}

func parseArgs() (config, error) {
	url := flag.String("url", "amqp://localhost:5672/", "URL to connect to")
	exchange := flag.String("exchange", "lenkung", "Exchange(s) to bind to (comma separated)")
	key := flag.String("key", "#", "Routing key to use in queue binding")
	port := flag.Int("port", 9090, "TCP port web UI")
	buf := flag.Int("buf", 100000, "Number of messages kept in memory")
	maxresult := flag.Int("maxresult", 1000, "Max. number of messages returned for query")
	pprint := flag.Bool("pprint", false, "Pretty-print JSON message body")
	flag.Parse()
	if len(flag.Args()) != 0 {
		return config{}, fmt.Errorf("Unexpected arguments %v", flag.Args())
	}
	var exchanges []string
	if strings.Contains(*exchange, ",") {
		exchanges = strings.Split(*exchange, ",")
	} else {
		exchanges = append(exchanges, *exchange)
	}
	return config{url: *url, exchanges: exchanges, key: *key, port: *port, bufferSize: *buf, maxResults: *maxresult, prettyPrint: *pprint}, nil
}

func main() {
	// profOut, err := os.Create("mqwatch.prof")
	// if err != nil {
	// 	log.Fatal("Could not create profiling file mqwatch.prof")
	// }
	// pprof.StartCPUProfile(profOut)
	// defer pprof.StopCPUProfile()
	cfg, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	}
	msgs := make(chan amqp.Delivery, 100)
	if err != nil {
		log.Fatal("Could not consume", err)
	}
	querych := make(chan query)
	ctrlch := make(chan string)
	dumpch := make(chan chan []message)
	channels := receiverChannels{querych, msgs, ctrlch, dumpch}
	go receive(channels, cfg)
	http.HandleFunc("/", queryHandler(cfg, channels))
	log.Printf("Listening on :%d, exchanges %v, routing key \"%s\"\n", cfg.port, cfg.exchanges, cfg.key)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.port), nil))
}
