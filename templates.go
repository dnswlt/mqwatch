package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"time"
)

const (
	indexHtml = `<!doctype html>
<html>
<head>
  <meta charset="UTF-8">
  <title>mqwatch</title>
  <style type="text/css">
body {
  font-family: Arial, sans-serif;
  font-size: 10pt;
}
.message .content {
  background-color: lightyellow;
  font-family: Courier New, monospace;
}
.message .routingkey {
  font-family: Courier New, monospace;
	background-color: lightblue;
}
  </style>
</head>
<body>
  Timestamp: {{.Created}}
  <h1>Frequencies</h1>
  <table>
  <tr>
    <th>Routing Key</th><th>Message Count</th>
  </tr>
  {{range $k, $v := .Frequencies}}
  <tr>
    <td>{{$k}}</td>
    <td>{{$v}}</td>
  </tr>
  {{end}}
  </table>

  <h1>Messages</h1>
  {{range .Messages}}
  <div class="message">
    <span class="received">{{DateFmt .Received}}</span> <span class="routingkey">{{.RoutingKey}}</span>
    <div class="content"><pre>{{MessageFmt .Body}}</pre></div>
  </div>
  {{end}}
</body>
</html>
`
)

func MessageFmt(bs []byte) string {
	var out bytes.Buffer
	json.Indent(&out, bs, "", "  ")
	return out.String()
}

func DateFmt(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func TemplateIndexHtml() *template.Template {
	return template.Must(template.New("index.html").
		Funcs(template.FuncMap{"MessageFmt": MessageFmt, "DateFmt": DateFmt}).
		Parse(indexHtml))
}
