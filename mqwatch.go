package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
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
	ClassName  string
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
}

// config contains all configurable parameters of the application
type config struct {
	url        string
	exchange   string
	key        string
	port       int
	bufferSize int
	maxResults int
}

func parseSpec(input string) []querySpec {
	splt := regexp.MustCompile("\n|,")
	ws := regexp.MustCompile(" +")
	exp := regexp.MustCompile(`(\w+):(.*)|#(\d+)?-(\d+)?`)
	var result []querySpec
	for _, line := range splt.Split(input, -1) {
		spec := querySpec{seqTo: math.MaxInt64}
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
	if !strings.Contains(m.RoutingKey, spec.routingKey) {
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

func receive(reqs <-chan query, msgs <-chan amqp.Delivery, cfg config) {
	var buf []message
	var seq int64
	maxBuf := int(float64(cfg.bufferSize) * 1.2)
	for {
		select {
		case msg := <-msgs:
			var m interface{}
			if msg.RoutingKey == "betriebsabbild.lenkereignisse.update" {
				break
			}
			err := json.Unmarshal(msg.Body, &m)
			if err != nil {
				log.Printf("Could not Unmarshal: %s\n\"%s\"\n", string(msg.Body), err)
				break
			}
			js, err := json.Marshal(m)
			if err != nil {
				log.Fatal("Could not marshal", err)
			}
			className, _ := msg.Headers["__ClassName__"].(string)
			buf = append(buf, message{seq, js, msg.RoutingKey, time.Now(), className})
			seq++
			l := len(buf)
			if l > maxBuf {
				buf = buf[l-cfg.bufferSize:]
			}
		case q := <-reqs:
			log.Printf("Processing query: %s\n", q.rawQuery)
			var r []message
			if string(q.rawQuery) == "*" {
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
			}
			q.respch <- queryResult{messages: r, seq: seq}
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

func queryHandler(querych chan<- query) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			return
		}
		reqStr := r.URL.Query().Get("q")
		var result queryResult
		if len(reqStr) < 3 && reqStr != "*" {
			log.Printf("Request string too short: %s\n", reqStr)
		} else {
			respch := make(chan queryResult)
			querych <- query{reqStr, parseSpec(reqStr), respch}
			result = <-respch
		}
		t := templateIndexHTML()
		w.Header().Set("Content-Type", "text/html")
		t.Execute(w, indexHTMLContent{
			Created:       time.Now(),
			Frequencies:   frequencies(result.messages),
			Messages:      result.messages,
			Query:         reqStr,
			ReceivedTotal: result.seq})
	}
}

func parseArgs() config {
	url := flag.String("url", "amqp://localhost:5672/", "URL to connect to")
	exchange := flag.String("exchange", "lenkung", "Exchange to bind to")
	key := flag.String("key", "#", "Routing key to use in queue binding")
	port := flag.Int("port", 9090, "TCP port web UI")
	buf := flag.Int("buf", 100000, "Number of messages kept in memory")
	maxresult := flag.Int("maxresult", 1000, "Max. number of messages returned for query")
	flag.Parse()
	return config{url: *url, exchange: *exchange, key: *key, port: *port, bufferSize: *buf, maxResults: *maxresult}
}

func main() {
	cfg := parseArgs()
	conn, err := amqp.Dial(cfg.url)
	if err != nil {
		log.Fatal("Could not connect", err)
	} else {
		log.Printf("Connected to %s\n", cfg.url)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Could not get a channel", err)
	}
	defer ch.Close()
	err = ch.ExchangeDeclare(cfg.exchange, "topic", false, false, false, false, nil)
	if err != nil {
		log.Fatal("Could not declare exchange", err)
	}
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		log.Fatal("Could not declare queue", err)
	}
	err = ch.QueueBind(q.Name, cfg.key, "lenkung", false, nil)
	if err != nil {
		log.Fatal("Could not bind queue", err)
	}
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("Could not consume", err)
	}
	querych := make(chan query)
	go receive(querych, msgs, cfg)
	http.HandleFunc("/", queryHandler(querych))
	log.Printf("Listening on :%d\n", cfg.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.port), nil))
}
