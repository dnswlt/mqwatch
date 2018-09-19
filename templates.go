package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/streadway/amqp"
)

// MessageFmt formats a RabbitMQ message by returing it as-is.
func MessageFmt(bs []byte) string {
	return string(bs)
}

// MessageFmtIndented formats a RabbitMQ JSON message by indenting it.
func MessageFmtIndented(bs []byte) string {
	var out bytes.Buffer
	err := json.Indent(&out, bs, "", "  ")
	if err != nil {
		return string(bs)
	}
	return out.String()
}

// DateFmt formats a time.Time as an ISO date.
func DateFmt(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

// HeaderFmt formats the header attributes
func HeaderFmt(header amqp.Table) string {
	var keys []string
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&out, "%s: %v\n", k, header[k])
	}

	return out.String()
}
