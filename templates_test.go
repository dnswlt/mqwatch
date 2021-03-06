package main

import (
	"bytes"
	"testing"
	"time"
)

func TestIndexHtml(t *testing.T) {
	tpl := templateIndexHTML(false)
	tm := time.Date(2018, 1, 31, 18, 59, 30, 123456789, time.Local)
	buf := new(bytes.Buffer)
	headers := make(map[string]interface{})
	/*
		Seq        int64
		Body       []byte
		RoutingKey string
		Received   time.Time
		Sender     string
		Headers    amqp.Table*/
	err := tpl.Execute(buf, indexHTMLContent{
		Created:     time.Now(),
		Frequencies: map[string]int{"a": 1, "b": 2},
		Exchanges:   []string{"exchange1", "exchange2"},
		Messages:    []message{message{0, []byte(`{"a": 100}`), "routing.key", tm, "sender", headers}},
		Query:       "abcde"})
	if err != nil {
		t.Error("Failed to write template", err)
	}
}
