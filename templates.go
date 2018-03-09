package main

import (
	"bytes"
	"encoding/json"
	"time"
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
