package main

import "testing"

func TestTypeAssert(t *testing.T) {
	m := map[string]interface{}{"a": 1}
	_, ok := m["b"]
	if ok {
		t.Error("\"b\" contained in map")
	}
	_, ok = m["a"].(int)
	if !ok {
		t.Error("\"a\" not an int")
	}
	b, ok := m["b"].(string)
	if ok {
		t.Error("Missing value is type assertible to string")
	}
	if b != "" {
		t.Error("Missing value not empty string")
	}
}
