package main

import (
	"html/template"
	"time"
)

const (
	indexHTML = `<!doctype html>
<html>
<head>
  <meta charset="UTF-8">
  <title>mqwatch</title>
  <style type="text/css">
body {
  font-family: Arial, sans-serif;
  font-size: 10pt;
}
tr {
  text-align: left;
}
.message .content {
  font-family: Courier New, monospace;
  font-size: 9pt;
}
.message .header {
	background-color: #b3e6ff;
}
  </style>
</head>
<body>
  Timestamp: {{DateFmt .Created}}, Total messages received: {{.ReceivedTotal}}
	<h1>Query</h1>
	<form method="get" action="/">
	<input type="text" name="q" size="80" value="{{.Query}}">
	<input type="submit" value="Query">
	</form>
	{{if .Frequencies}}
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
	{{end}}

	{{if .Messages}}
  <h1>Messages</h1>
  {{range .Messages}}
  <div class="message">
    <div class="header">{{DateFmt .Received}} ({{.Sender}}@{{DateFmt .Sent}}) (#{{.Seq}}) {{.RoutingKey}} {{.ClassName}}</div>
    <div class="content"><pre>{{MessageFmt .Body}}</pre></div>
  </div>
  {{end}}
	{{else}}
	<p>No messages.</p>
	{{end}}
</body>
</html>
`
)

type indexHTMLContent struct {
	Created       time.Time
	Frequencies   map[string]int
	Messages      []message
	Query         string
	ReceivedTotal int64
}

// templateIndexHTML returns the template for the index.html page.
func templateIndexHTML() *template.Template {
	return template.Must(template.New("index.html").
		Funcs(template.FuncMap{"MessageFmt": MessageFmt, "DateFmt": DateFmt}).
		Parse(indexHTML))
}
