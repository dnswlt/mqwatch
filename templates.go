package main

import (
	"bytes"
	"encoding/json"
	"sort"
	"time"

	"github.com/streadway/amqp"
)

// MessageFmt formats a RabbitMQ JSON message by indenting it.
func MessageFmt(bs []byte) string {
	var out bytes.Buffer
	json.Indent(&out, bs, "", "  ")
	return out.String()
}

// DateFmt formats a time.Time as an ISO date.
func DateFmt(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

// Formats the header attributes
func HeaderFmt(header amqp.Table) string {
	var keys []string
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out bytes.Buffer
	for _, k := range keys {
		out.WriteString(k)
		out.WriteString(" = ")
		value, err := json.Marshal(header[k])
		if err != nil {
			out.WriteString("Error printing value: ")
			out.WriteString(err.Error())
		} else {
			out.WriteString(string(value))
		}
		out.WriteString("\n")
	}

	return out.String()
}