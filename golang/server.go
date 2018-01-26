package golang

import (
	"bytes"
	"fmt"
	"go/format"
	"html/template"
	"net/url"
	"sort"
	"strings"

	"github.com/tfkhsr/jsonmsg"
	"github.com/tfkhsr/jsonschema"
	"github.com/tfkhsr/jsonschema/golang"
)

// Generates go src for a server from a jsonmsg.Spec without imports and package
func ServerSrc(s *jsonmsg.Spec) ([]byte, error) {
	ifc, err := generateInterfaceType(s)
	if err != nil {
		return nil, err
	}

	outs, err := generateOutTypes(s)
	if err != nil {
		return nil, err
	}

	hlp, err := generateHelper(s)
	if err != nil {
		return nil, err
	}

	httph, err := generateHTTPHandler(s)
	if err != nil {
		return nil, err
	}

	idx, err := jsonschema.Parse(s.Raw)
	if err != nil {
		return nil, err
	}

	typ, err := golang.Src(idx)
	if err != nil {
		return nil, err
	}

	espc, err := generateEmbeddedJSONSpec(s)
	if err != nil {
		return nil, err
	}

	ehspc, err := generateEmbeddedHTMLSpec(s)
	if err != nil {
		return nil, err
	}

	w := &bytes.Buffer{}
	fmt.Fprintf(w, "%s", ifc)
	fmt.Fprintf(w, "%s", outs)
	fmt.Fprintf(w, "%s", hlp)
	fmt.Fprintf(w, "%s", httph)
	fmt.Fprintf(w, "%s", typ)
	fmt.Fprintf(w, "%s", espc)
	fmt.Fprintf(w, "%s", ehspc)

	return format.Source(w.Bytes())
}

// Generates go src for a server from a jsonmsg.Spec as a complete package with imports
func ServerPackageSrc(s *jsonmsg.Spec, pack string) ([]byte, error) {
	src, err := ServerSrc(s)
	if err != nil {
		return nil, err
	}

	w := &bytes.Buffer{}
	fmt.Fprintf(w, `package %v

import (
`, pack)
	for _, i := range ServerImports(src) {
		fmt.Fprintf(w, "\t\"%s\"\n", i)
	}
	fmt.Fprintf(w, ")\n%s", src)

	return format.Source(w.Bytes())
}

// Returns a list of required imports
func ServerImports(src []byte) []string {
	return unionStrings(
		golang.Imports(src),
		importsForSrc(src),
	)
}

func generateInterfaceType(s *jsonmsg.Spec) ([]byte, error) {
	w := bytes.NewBufferString("\n")
	fmt.Fprintf(w, "type API interface {\n")
	var keys []string
	for k, _ := range s.Messages {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		m := s.Messages[k]
		if m.InSchema != nil {
			if len(m.OutSchemas) > 0 {
				fmt.Fprintf(w, "\t%s(*%s) (*%vOuts, error)\n", m.Name, m.InSchema.Name, m.Name)
			} else {
				fmt.Fprintf(w, "\t%s(*%s) error\n", m.Name, m.InSchema.Name)
			}
		} else {
			if len(m.OutSchemas) > 0 {
				fmt.Fprintf(w, "\t%s() (*%vOuts, error)\n", m.Name, m.Name)
			} else {
				fmt.Fprintf(w, "\t%s() error\n", m.Name)
			}
		}
	}
	fmt.Fprintf(w, "}\n")

	return format.Source(w.Bytes())
}

func generateOutTypes(s *jsonmsg.Spec) ([]byte, error) {
	w := bytes.NewBufferString("\n")
	var keys []string
	for k, _ := range s.Messages {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		m := s.Messages[k]
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "type %vOuts struct {\n", m.Name)
		for _, o := range m.OutSchemas {
			fmt.Fprintf(w, "  %v *%v\n", o.Name, o.Name)
		}
		fmt.Fprintf(w, "}\n")
	}

	return format.Source(w.Bytes())
}

// Generates an HTTP and HTTPS handler
func generateHTTPHandler(s *jsonmsg.Spec) ([]byte, error) {
	tmpl, err := template.New("handler").Funcs(template.FuncMap{
		"Contains": stringsContain,
		"ParseURL": func(rawurl string) *url.URL {
			u, err := url.Parse(rawurl)
			if err != nil {
				panic(err)
			}
			return u
		},
	}).Parse(httpHandlerTemplate)
	if err != nil {
		return nil, err
	}

	w := bytes.NewBufferString("\n")
	err = tmpl.Execute(w, s)
	if err != nil {
		return nil, err
	}

	return format.Source(w.Bytes())
}

// Generates helper
func generateHelper(s *jsonmsg.Spec) ([]byte, error) {
	w := bytes.NewBufferString("\n")
	fmt.Fprintf(w, `
// jsonmsg message schema
type message struct {
	Msg string 
	Data json.RawMessage
}

func newMessageReader(msg string, data interface{}) io.Reader {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	m := &message{msg, b}
	raw, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes.NewBuffer(raw)
}

type outMessage struct {
	Msg string
	Data interface{}
}

func newValueMessage(msg string, data interface{}) outMessage {
	return outMessage{
		Msg: msg,
		Data: data,
	}
}

type errorData struct {
	Error string
}

func newErrorMessage(e string) outMessage {
	return outMessage{
		Msg: "error",
		Data: errorData{e},
	}
}

var (
	UnparsableRequestErrorMessage = newErrorMessage("unparsable message")
	InternalErrorMessage          = newErrorMessage("internal error")
	UnknownMessageErrorMessage    = newErrorMessage("unknown message")
	InvalidMethodErrorMessage     = newErrorMessage("only POST method allowed")
)
`)

	// json annotations (unfortunately not possible in multiline strings)
	src := w.String()
	src = strings.Replace(src, "Msg string", "Msg string `json:\"msg\"`", -1)
	src = strings.Replace(src, "Data json.RawMessage", "Data json.RawMessage `json:\"data\"`", -1)
	src = strings.Replace(src, "Data interface{}", "Data interface{} `json:\"data\"`", -1)
	src = strings.Replace(src, "Error string", "Error string `json:\"error\"`", -1)

	return format.Source([]byte(src))
}

// Generates embedded spec
func generateEmbeddedJSONSpec(s *jsonmsg.Spec) ([]byte, error) {
	raw, err := s.JSONSpec()
	if err != nil {
		return nil, err
	}

	w := bytes.NewBufferString("\n")
	fmt.Fprintf(w, "// embedded spec\n")
	fmt.Fprintf(w, "func newEmbeddedSpec() []byte {\n")
	fmt.Fprintf(w, "\treturn []byte(`%s`)\n", raw)
	fmt.Fprintf(w, "}\n")

	return format.Source(w.Bytes())
}

// Generates embedded html spec
func generateEmbeddedHTMLSpec(s *jsonmsg.Spec) ([]byte, error) {
	w := bytes.NewBufferString("\n")
	fmt.Fprintf(w, "// embedded html spec\n")
	fmt.Fprintf(w, "func newEmbeddedHTMLSpec() []byte {\n")
	fmt.Fprintf(w, "\treturn []byte(`\n")

	h, err := s.HTTPSpec()
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(w, "%s", h)

	fmt.Fprintf(w, "\t`)\n")
	fmt.Fprintf(w, "}\n")

	return format.Source(w.Bytes())
}

// Returns a list of required Go imports
func importsForSrc(src []byte) []string {
	i := []string{
		"net/http",
		"encoding/json",
		"bytes",
		"io",
		"io/ioutil",
	}

	// websockets
	if strings.Contains(string(src), "websocket.Upgrader") {
		i = append(i, "github.com/gorilla/websocket")
	}

	sort.Strings(i)
	return i
}

const httpHandlerTemplate = `
func NewAPIMux(i API) *http.ServeMux {
  mux := http.NewServeMux()

	// processing logic
	processMessage := func(in []byte) (interface{}, int) {
		var err error

		// parse message
		var m message
		err = json.Unmarshal(in, &m)
		if err != nil {
			return UnparsableRequestErrorMessage, http.StatusUnprocessableEntity
		}

		// fetchSpec
		if m.Msg == "fetchSpec" {
			return newValueMessage("spec", json.RawMessage(newEmbeddedSpec())), http.StatusOK
		}
		
		{{ range .Messages }} 
		// {{ .Name }}
		if m.Msg == "{{ .Msg }}" {

			{{ if .InSchema }}
			// parse data
			var data {{ .InSchema.Name }}
			err = json.Unmarshal(m.Data, &data)
			if err != nil {
				return UnparsableRequestErrorMessage, http.StatusUnprocessableEntity
			}

			err = data.Validate()
			if err != nil {
				return newErrorMessage(err.Error()), http.StatusUnprocessableEntity
			}
			{{ end }}

			// dispatch message
			{{ if .InSchema }}
				{{ if .OutSchemas }}
			outs, err := i.{{ .Name }}(&data)
				{{ else }}
			err = i.{{ .Name }}(&data)
				{{ end }}
			{{ else }}
				{{ if .OutSchemas }}
			outs, err := i.{{ .Name }}()
				{{ else }}
			err = i.{{ .Name }}()
				{{ end }}
			{{ end }}
			if err != nil {
				return InternalErrorMessage, http.StatusInternalServerError
			}

			{{ if .OutSchemas }}
			// handler returned nothing
			if outs == nil {
				return InternalErrorMessage, http.StatusInternalServerError
			}

			// select the first non-nil out
			{{ range .OutSchemas }}
			if outs.{{ .Name }} != nil {
				err = outs.{{ .Name }}.Validate()
				if err != nil {
					return InternalErrorMessage, http.StatusInternalServerError
				}
				return newValueMessage("{{ .JSONName }}", outs.{{ .Name }}), http.StatusOK
			}
			{{ end }}
				
			// no outs and no error
			return InternalErrorMessage, http.StatusInternalServerError
			{{ else }}
			return nil, http.StatusOK
			{{ end }}
		}
		{{ end }}
		
		// unknown msg
		return UnknownMessageErrorMessage, http.StatusNotFound
	}

	// GET /spec.json
  mux.HandleFunc("{{ .Endpoints.BasePath }}/spec.json", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		
		// headers
		w.Header().Set("Content-Type", "application/json")

		// ensure GET
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			enc.Encode(InvalidMethodErrorMessage)
			return
		}

		w.WriteHeader(http.StatusOK)
		enc.Encode(json.RawMessage(newEmbeddedSpec()))
		return
	})
	
	// GET /spec
  mux.HandleFunc("{{ .Endpoints.BasePath }}/spec", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		
		// headers
		w.Header().Set("Content-Type", "text/html")

		// ensure GET
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			enc.Encode(InvalidMethodErrorMessage)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(newEmbeddedHTMLSpec())
		return
	})

	{{ if (index .Endpoints.URLs "http") }}
	// protocol: http
	// POST /http
  mux.HandleFunc("{{ (ParseURL (index .Endpoints.URLs "http")).EscapedPath }}", func(w http.ResponseWriter, r *http.Request) {
		var err error
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers")
		w.Header().Set("Access-Control-Allow-Methods", "POST")

		// handle OPTIONS
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// ensure POST
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			enc.Encode(InvalidMethodErrorMessage)
			return
		}
		
		// parse message
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		// process message
		out, statusCode := processMessage(body)
		w.WriteHeader(statusCode)
		enc.Encode(out)
	})
	{{ end }}
	
	
	{{ if (index .Endpoints.URLs "websocket") }}
	// protocol: websocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: nil,
	}

	// GET /websocket
  mux.HandleFunc("{{ (ParseURL (index .Endpoints.URLs "websocket")).EscapedPath }}", func(w http.ResponseWriter, r *http.Request) {
		var err error
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc.Encode(InternalErrorMessage)
			return
		}
	
		// read/write loop
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
		
			// process message
			out, _ := processMessage(data)
			outMsg, err := json.MarshalIndent(out, "", "  ")
			if err == nil {
				conn.WriteMessage(websocket.TextMessage, outMsg)
			}
		}
	})
	{{ end }}

	return mux
}
`

// Returns the union of string slices
func unionStrings(s ...[]string) []string {
	m := make(map[string]bool)
	for _, ss := range s {
		for _, k := range ss {
			m[k] = true
		}
	}

	u := []string{}
	for k, _ := range m {
		u = append(u, k)
	}
	sort.Strings(u)
	return u
}

// Checks if a slice of strings contains a string
func stringsContain(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
