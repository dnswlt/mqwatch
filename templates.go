package main

import (
	"html/template"
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
    <span class="received">{{.Received}}</span> <span class="routingkey">{{.RoutingKey}}</span>
    <div class="content">{{String .Body}}</div>
  </div>
  {{end}}
</body>
</html>
`
)

func TemplateIndexHtml() *template.Template {
	return template.Must(template.New("index.html").
		Funcs(template.FuncMap{"String": func(bs []byte) string { return string(bs) }}).
		Parse(indexHtml))
}
