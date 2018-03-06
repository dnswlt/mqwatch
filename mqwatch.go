package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/streadway/amqp"
)

const (
	MAX_BUF = 100000
)

type message struct {
	body       []byte
	routingKey string
	received   time.Time
}
type query struct {
	text   []byte
	respch chan []message
}

func receive(reqs <-chan query, msgs <-chan amqp.Delivery) {
	var buf []message
	for {
		select {
		case msg := <-msgs:
			var m interface{}
			err := json.Unmarshal(msg.Body, &m)
			if err != nil {
				log.Printf("Could not Unmarshal: %s\n\"%s\"\n", string(msg.Body), err)
				break
			}
			js, err := json.Marshal(m)
			if err != nil {
				log.Fatal("Could not marshal", err)
			}
			buf = append(buf, message{js, msg.RoutingKey, time.Now()})
			l := len(buf)
			if l > MAX_BUF*1.2 {
				buf = buf[l-MAX_BUF:]
			}
		case q := <-reqs:
			log.Printf("Received request: %s\n", string(q.text))
			var r []message
			for _, m := range buf {
				if bytes.Contains(m.body, q.text) {
					r = append(r, m)
				}
			}
			q.respch <- r
		}
	}
}

func frequencies(ms []message) map[string]int {
	freq := make(map[string]int)
	for _, m := range ms {
		cnt, ok := freq[m.routingKey]
		if ok {
			freq[m.routingKey] = cnt + 1
		} else {
			freq[m.routingKey] = 1
		}
	}
	return freq
}

func queryHandler(querych chan<- query) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reqStr := r.URL.Query().Get("s")
		if len(reqStr) < 3 {
			log.Printf("Request string too short: %s\n", reqStr)
			return
		}
		respch := make(chan []message)
		querych <- query{[]byte(reqStr), respch}
		ms := <-respch
		w.Header().Set("Content-Type", "text/html")
		t, err := template.ParseFiles("templates/index.html")
		fmt.Fprintf(w, "Found %d items\n", len(ms))
		fmt.Fprintf(w, "Routing key frequencies:\n")
		for rk, freq := range frequencies(ms) {
			fmt.Fprintf(w, "\t%s: %d\n", rk, freq)
		}
		for _, m := range ms {
			fmt.Fprintf(w, "Item %s on %s:\n\t%s\n", m.received.Format(time.RFC3339Nano), m.routingKey, string(m.body))
		}
	}
}

func main() {
	conn, err := amqp.Dial("amqp://localhost:5672/")
	if err != nil {
		log.Fatal("Could not connect", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Could not get a channel", err)
	}
	defer ch.Close()
	// func (ch *Channel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args Table) error {
	err = ch.ExchangeDeclare("lenkung", "topic", false, false, false, false, nil)
	if err != nil {
		log.Fatal("Could not declare exchange", err)
	}
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		log.Fatal("Could not declare queue", err)
	}
	err = ch.QueueBind(q.Name, "#", "lenkung", false, nil)
	if err != nil {
		log.Fatal("Could not bind queue", err)
	}
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("Could not consume", err)
	}
	querych := make(chan query)
	go receive(querych, msgs)

	http.HandleFunc("/", queryHandler(querych))
	log.Fatal(http.ListenAndServe(":9090", nil))
}
