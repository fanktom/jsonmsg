/*
Package jsonmsg parses API specs.
The resulting spec can be used to generate server/client source code in any supported language.

The jsonmsg implementation is based on https://github.com/jsonmsg/spec with the meta-schema https://github.com/jsonmsg/spec/blob/master/meta.json

Generators

go: https://godoc.org/github.com/tfkhsr/jsonmsg/golang


Parse a spec:

	schema := `
	{
	  "endpoints": {
	    "http": "https://jsonmsg.github.io/v1",
	    "websocket": "wss://jsonmsg.github.io/v1"
	  },
	  "messages": {
	    "findUser": {
	      "in": "#/definitions/userQuery",
	      "outs": [
	        "#/definitions/user",
	        "#/definitions/error"
	      ],
	      "group": "user"
	    }
	  },
	  "definitions": {
	    "user": {
	      "type": "object",
	      "properties": {
	        "id": {
	          "type": "string"
	        },
	        "name": {
	          "type": "string"
	        }
	      },
	      "required": ["id", "name"]
	    },
	    "userQuery": {
	      "type": "object",
	      "properties": {
	        "id": {
	          "type": "string"
	        }
	      },
	      "required": ["id"]
	    },
	    "error": {
	      "type": "object",
	      "properties": {
	        "message": {
	          "type": "string"
	        }
	      },
	    "required": ["message"]
	    }
	  }
	}
	`

	// parse spec
	spc, err := Parse(spec)
	if err != nil {
		panic(err)
	}

	// spc now contains:
	// spc.Messages["findUser"]        : *Message{...}
	// spc.Definitions["user"]         : *jsonschema.Schema{...}
*/
package jsonmsg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/tfkhsr/jsonschema"
)

// A Spec holds endpoints, messages and definitions of an API.
// Parsed spec must validate against meta-schema https://github.com/jsonmsg/spec/blob/master/meta.json
type Spec struct {
	// General title of API
	Title string

	// Further information
	Description string

	// Map of protocols to URLs (urlString embeds url.URL for unmarshaling)
	Endpoints map[string]*urlString

	// Map of message names to Messages
	Messages map[string]*Message

	// Map of groups of message names to Messages
	GroupedMessages map[string]map[string]*Message

	// JSON Schema definitions for data definition
	Definitions jsonschema.Index `json:"-"`

	// Raw spec
	Raw []byte
}

type urlString struct {
	url.URL
}

func (s *urlString) UnmarshalJSON(b []byte) error {
	us := string(b)
	u, err := url.Parse(us[1 : len(us)-1])
	if err != nil {
		return err
	}
	*s = urlString{*u}
	return nil
}

// A Message holds details about name, in- and out parameters
type Message struct {
	// Short description
	Title string

	// Detailed description
	Description string

	// Literal message name as defined in spec
	Msg string

	// Camel-Cased name
	Name string

	// JSON Pointer to input schema
	In string

	// JSON Pointers to possible output schemas
	Outs []string

	// Pointer to parsed input schema
	InSchema *jsonschema.Schema

	// Pointers to parsed output schemas
	OutSchemas []*jsonschema.Schema

	// Pointer to parent spec
	Spec *Spec

	// Optional: Group name associating message to spec.GroupedMessages
	Group string
}

// Parses a raw schema into a Spec
func Parse(b []byte) (*Spec, error) {
	var spec Spec
	err := json.Unmarshal(b, &spec)
	if err != nil {
		return nil, err
	}

	// raw
	spec.Raw = b

	// endpoints
	for k, _ := range spec.Endpoints {
		spec.Endpoints[k].Path += "/" + k
	}

	// definitions
	idx, err := jsonschema.Parse(b)
	if err != nil {
		return nil, err
	}
	spec.Definitions = *idx

	// messages
	spec.GroupedMessages = make(map[string]map[string]*Message)

	for k, _ := range spec.Messages {
		spec.Messages[k].Msg = k
		spec.Messages[k].Name = goNameFromStrings(k)
		spec.Messages[k].Spec = &spec

		InSchema, err := resolvePointerToSchema(spec.Messages[k].In, &spec.Definitions)
		if err != nil {
			return nil, err
		}
		spec.Messages[k].InSchema = InSchema

		spec.Messages[k].OutSchemas = make([]*jsonschema.Schema, 0)
		for i, _ := range spec.Messages[k].Outs {
			outSchema, err := resolvePointerToSchema(spec.Messages[k].Outs[i], &spec.Definitions)
			if err != nil {
				return nil, err
			}
			spec.Messages[k].OutSchemas = append(spec.Messages[k].OutSchemas, outSchema)
		}

		if _, ok := spec.GroupedMessages[spec.Messages[k].Group]; !ok {
			spec.GroupedMessages[spec.Messages[k].Group] = make(map[string]*Message)
		}
		spec.GroupedMessages[spec.Messages[k].Group][k] = spec.Messages[k]
	}

	return &spec, nil
}

// Returns an HTML website version of the spec that must be served on {{ BaseURL }}/spec by servers
func (s *Spec) HTTPSpec() ([]byte, error) {
	w := &bytes.Buffer{}
	err := writeTemplate(s, httpSpecTemplate, w)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// Returns an indented, marshalled version of the spec that must be served on {{ BaseURL }}/spec.json by servers
func (s *Spec) JSONSpec() ([]byte, error) {
	var o interface{}
	err := json.Unmarshal(s.Raw, &o)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(o, "", "  ")
}

// Creates a new message instance conforming to the message schema
func (m *Message) NewInstance() (interface{}, error) {
	nm := make(map[string]interface{})
	nm["msg"] = m.Msg
	if m.InSchema == nil {
		return nm, nil
	}
	data, err := m.InSchema.NewInstance(&m.Spec.Definitions)
	if err != nil {
		return nil, err
	}
	nm["data"] = data
	return nm, nil
}

// creates a go friendly name from string parts
func goNameFromStrings(parts ...string) string {
	name := ""
	re := regexp.MustCompile("{|}|-|_")
	for _, p := range parts {
		c := re.ReplaceAllString(p, "")
		switch c {
		case "id":
			name += "ID"
		case "url":
			name += "URL"
		case "api":
			name += "API"
		default:
			name += strings.Title(c)
		}
	}
	return name
}

// returns a schema or referenced schema
func resolvePointerToSchema(p string, idx *jsonschema.Index) (*jsonschema.Schema, error) {
	if p == "" {
		return nil, nil
	}
	s, ok := (*idx)[p]
	if !ok {
		return nil, fmt.Errorf("jsonmsg: %v does not exist in index", p)
	}
	return s, nil
}

// Parses the spec input, applies the template and writes it to the writer
func writeTemplate(s *Spec, t string, w io.Writer) error {
	tmpl, err := template.New("spec").Funcs(template.FuncMap{
		"Contains": stringsContain,
		"Add": func(a int, b int) int {
			return a + b
		},
		"JSON": func(in interface{}) string {
			jsn, err := json.MarshalIndent(in, "", "  ")
			if err != nil {
				panic(err)
			}
			return string(jsn)
		},
		"SubstringRight": func(a string, n int) string {
			return a[0 : len(a)-n]
		},
	}).Parse(t)
	if err != nil {
		return err
	}
	err = tmpl.Execute(w, s)
	if err != nil {
		return err
	}

	return nil
}

const httpSpecTemplate = `
<!DOCTYPE html>
<html>
  <head>
    <title>{{ .Title }}</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
		<style>
			body {
				font-family: Arial, sans-serif;
    		color: #333333;
				margin: 1.5em;
			}

			h1 {
    		color: #3b4151;
			}

			h2 {
    		color: #3b4151;
				padding-top: 1em;
				font-size: 1.2em;
			}

			h3 {
    		color: #3b4151;
    		background: rgba(97,175,254,.1);
				padding: 1em;
				margin-top: 2em;
			}

			h3 span {
				padding-left: 1em;
				font-size: 1em;
				font-weight: normal;
			}
			
			h4 {
    		color: #555d73;
				padding: 1em;
				margin: 0;
				margin-top: 2em;
			}

			dd {
				margin-left: 1em;
				line-height: 1.4em;
			}
			dd.level1 {
				margin-left: 2em;
			}
			
			dd.level2 {
				margin-left: 3em;
			}

			a {
    		color: #3b4151;
    		text-decoration: none;
			}

			a:hover {
				text-decoration: underline;
			}

			p.level1 {
				padding-left: 1em;
				padding-right: 1em;
			}
			
			p.level2 {
				padding-left: 2em;
				padding-right: 1em;
			}

			.row {
				display: flex;
				max-width: 50em;
			}

			.row.right {
				justify-content: flex-end;
			}

			.row div {
				display: flex;
				padding: 1em;
			}

			.row button {
				margin-top: 1em;
				margin-bottom: 1em;
				margin-left: 1em;
			}

			span.property {
				font-weight: bold;
			}
			
			span.type {
    		color: #3b4151;
			}
			
			span.required {
    		color: #737373;
				padding-left: 0.5em;
			}

			pre.code,
			textarea.code {
				width: 100%;
				border: none;
				background: #EFEFEF;
    		padding: 1em;
				overflow-y: scroll;
				margin: 0;
				margin-left: 1em;
				color: inherit;
			}

			pre.invisible {
				display: none;
			}

			pre.bg-success {
				background: rgba(97, 254, 153, 0.29);
			}
			
			pre.bg-error {
				background: rgba(254, 97, 97, 0.18);
			}
		</style>
		<script>
			var jsonmsg = {
				{{ if (index .Endpoints "http") }}
				http: {
					endpoint: "{{ js (index .Endpoints "http").String }}",
					send: function(msg) {
						return fetch(jsonmsg.http.endpoint, {
							method: "POST",
							headers: {
								"Content-Type": "application/json"
							},
							body: JSON.stringify(msg)
						})
						.then(function(response) {
  						return response.json();
						});
					},
					sendFromInputToOutput: function(inId, outId) {
						var msg = document.querySelector(inId).value;
						var out = document.querySelector(outId);

						out.innerHTML = "sending ...";

						jsonmsg.http.send(JSON.parse(msg))
						.then(function(m){
							out.innerHTML = JSON.stringify(m, null, 2);
							out.classList = "code bg-success";
						})
						.catch(function(e) {
							out.innerHTML = e;
							out.classList = "code bg-error";
						});
					}
				},
				{{ end }}
				
				{{ if (index .Endpoints "websocket") }}
				websocket: {
					endpoint: "{{ js (index .Endpoints "websocket").String }}",
					conn: undefined,
					connect: function() {
						jsonmsg.websocket.conn = new WebSocket(jsonmsg.websocket.endpoint);
					},
					send: function(msg) {
						if(!jsonmsg.websocket.conn) {
							jsonmsg.websocket.connect();
						}
						if (jsonmsg.websocket.conn.readyState !== 1) {
							jsonmsg.websocket.conn.addEventListener("open", function(){
								jsonmsg.websocket.conn.send(JSON.stringify(msg));
							}, { once: true });
						} else {
							jsonmsg.websocket.conn.send(JSON.stringify(msg));
						}
					},
					sendFromInputToOutput: function(inId, outId) {
						var msg = document.querySelector(inId).value;
						var out = document.querySelector(outId);

						out.innerHTML = "sending ...";
						
						if(!jsonmsg.websocket.conn) {
							jsonmsg.websocket.connect();
						}
						jsonmsg.websocket.conn.addEventListener("message", function(m){
							out.innerHTML = m.data;
							out.classList = "code bg-success";
						}, { once: true });
						jsonmsg.websocket.send(JSON.parse(msg));
					}
				},
				{{ end }}
			};
		</script>
  </head>
  <body>
		<h1>{{ .Title }}</h1>
		<p>{{ .Description }}</p>

		<a name="index"></a>
		<h2>Index</h2>
		<dl>
		<dd><a href="#specs">Specs</a></dd>
		<dd class="level1"><a href="#spec-json">json</a></dd>
		<dd class="level1"><a href="#spec-html">html</a></dd>
		<dd><a href="#endpoints">Endpoints</a></dd>
		{{ range $k, $v := .Endpoints }}
		<dd class="level1"><a href="#endpoint-{{ $k }}">{{ $k }}</a></dd>
		{{ end }}

		<dd><a href="#messages">Messages</a></dd>
		{{ range $g, $ms := .GroupedMessages }}
		<dd class="level1"><a href="#group-{{ $g }}">{{ $g }}</a></dd>
		{{ range $k, $v := $ms }}
		<dd class="level2"><a href="#message-{{ $k }}">{{ $k }}</a></dd>
		{{ end }}
		{{ end }}

		<dd><a href="#data">Data</a></dd>
		{{ range $k, $v := .Definitions }}
		{{ if or (eq $v.Type "object") (eq $v.Type "array") }}
		<dd class="level1"><a href="#data-{{ $v.PointerName }}">{{ $v.PointerName }}</a></dd>
		{{ end }}
		{{ end }}

		</dl>
		
		<a name="specs"></a>
		<h2>Specs</h2>
		<a name="spec-json"></a>
		<h3>
			json
			<span><a href="{{ SubstringRight .Endpoints.http.String 5 }}/spec.json">{{ SubstringRight .Endpoints.http.String 5 }}/spec.json</a></span>
		</h3>
		<p class="level1">Machine readable spec for API</p>

		<a name="spec-html"></a>
		<h3>
			html
			<span><a href="{{ SubstringRight .Endpoints.http.String 5 }}/spec">{{ SubstringRight .Endpoints.http.String 5 }}/spec</a></span>
		</h3>
		<p class="level1">Human readable spec for API</p>

		<a name="endpoints"></a>
		<h2>Endpoints</h2>
		{{ range $k, $v := .Endpoints }}
		<a name="endpoint-{{ $k }}"></a>
		<h3>
			{{ $k }}
			<span>{{ $v.String }}</span>
		</h3>
		{{ end }}
		
		<a name="messages"></a>
		<h2>Messages</h2>
		{{ range $g, $ms := .GroupedMessages }}
		<a name="group-{{ $g }}"></a>
		{{ range $k, $v := $ms }}
		<a name="message-{{ $k }}"></a>
		<h3>
		{{ $k }}
		{{ if $g }}<span>{{ $g }}</span>{{ end }}
		<span>{{ $v.Title }}</span>
		</h3>
		<p class="level1">{{ $v.Description }}</p>
		<dl>
		<dd>In</dd>
		{{ if $v.InSchema }}
		<dd class="level1"><a href="#data-{{ $v.InSchema.PointerName }}">{{ $v.InSchema.PointerName }}</a></dd>
		{{ else }}
		<dd class="level1">None</dd>
		{{ end }}
		<dd>Outs</dd>
		{{ range $v.OutSchemas }}
		<dd class="level1"><a href="#data-{{ .PointerName }}">{{ .PointerName }}</a></dd>
		{{ end }}
		</dl>

		<h4>Test</h4>
		<div class="row">
		<textarea class="code" id="input-{{ $k }}" spellcheck="false" rows={{ if not $v.InSchema }}5{{ else }}{{ Add 5 (len $v.InSchema.Properties) }}{{ end }}>{{ JSON $v.NewInstance }}</textarea>
		</div>
		<div class="row right">
			{{ if (index $.Endpoints "websocket") }}
			<button onclick="jsonmsg.websocket.sendFromInputToOutput('#input-{{ $k }}', '#output-{{ $k }}')">Send WebSocket</button>
			{{ end }}
			{{ if (index $.Endpoints "http") }}
			<button onclick="jsonmsg.http.sendFromInputToOutput('#input-{{ $k }}', '#output-{{ $k }}')">Send HTTP</button>
			{{ end }}
		</div>
		<div class="row">
		<pre class="code invisible" id="output-{{ $k }}"></pre>
		</div>
		{{ end }}
		{{ end }}

		<a name="data"></a>
		<h2>Data</h2>
		{{ range $k, $v := .Definitions }}
		{{ if or (eq $v.Type "object") (eq $v.Type "array") }}
		<a name="data-{{ $v.PointerName }}"></a>
		<h3>
			{{ $v.PointerName }}
			<span>{{ $v.Type }}</span>
			<span>{{ $v.Title }}</span>
		</h3>
		<p class="level1">{{ $v.Description }}</p>
		
		{{ if eq $v.Type "array" }}
		<div class="row">
			<div style="width: 33%"><span class="property">Items</span></div>
			<div style="width: 33%"><span class="type">
			{{ if eq $v.Items.Type "ref" }}
				<a href="#data-{{ (index $.Definitions $v.Items.Ref).PointerName }}">{{ (index $.Definitions $v.Items.Ref).PointerName }}</a>
			{{ else if or (eq $v.Type "object") (eq $v.Type "array") }}
				<a href="#data-{{ $v.PointerName }}">{{ $v.PointerName }}/items</a>
			{{ else }}
				{{ $v.Items.Type }}
			{{ end }}
			</span></div>
			<div style="width: 33%">{{ $v.Items.Description }}</div>
		</div>
		{{ end }}

		{{ if eq $v.Type "object" }}
		<h4>Properties</h4>
		{{ range $pk, $pv := $v.Properties }}
		<div class="row">
			<div style="width: 33%">
			<span class="property">{{ $pk }}</span>
			{{ if Contains $v.Required $pk }}
			<span class="required">(required)</span>
			{{ end }}
			</div>
			<div style="width: 33%"><span class="type">
			{{ if eq $pv.Type "ref" }}
				<a href="#data-{{ (index $.Definitions $pv.Ref).PointerName }}">{{ (index $.Definitions $pv.Ref).PointerName }}</a>
			{{ else if or (eq $pv.Type "object") (eq $pv.Type "array") }}
				<a href="#data-{{ $pv.PointerName }}">{{ $pv.PointerName}}</a>
			{{ else }}
				{{ $pv.Type }}
			{{ end }}
			</span></div>
			<div style="width: 33%">{{ $pv.Description }}</div>
		</div>
		{{ end }}
		{{ end }}

		{{ end }}

		{{ end }}

  </body>
</html>
`

// Checks if a slice of strings contains a string
func stringsContain(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
