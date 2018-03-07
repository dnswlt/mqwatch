package main

import (
	"bytes"
	"testing"
	"time"
)

func TestIndexHtml(t *testing.T) {
	tpl := TemplateIndexHtml()
	tm := time.Date(2018, 1, 31, 18, 59, 30, 123456789, time.Local)
	buf := new(bytes.Buffer)
	err := tpl.Execute(buf, indexContent{
		Created:     time.Now(),
		Frequencies: map[string]int{"a": 1, "b": 2},
		Messages:    []message{message{[]byte(`{"a": 100}`), "routing.key", tm}}})
	if err != nil {
		t.Error("Failed to write template", err)
	}
	s := buf.String()
	t.Log("UFFE", s)
}
