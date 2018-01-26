package golang

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/tfkhsr/jsonmsg"
	"github.com/tfkhsr/jsonmsg/fixture"
)

func TestGenerateGoInterfaceType(t *testing.T) {
	spc, err := jsonmsg.Parse([]byte(fixture.TestSchemaSimpleLogin))
	if err != nil {
		t.Fatal(err)
	}
	o := `
type API interface {
	LoginWithCredentials(*Credentials) (*LoginWithCredentialsOuts, error)
	Logout(*Session) (*LogoutOuts, error)
}
`
	typ, err := generateInterfaceType(spc)
	if err != nil {
		t.Fatal(err)
	}
	if string(typ) != o {
		t.Fatalf("type should be '%s' but is '%s'", o, typ)
	}
}

func TestGenerateGoHTTPHandler(t *testing.T) {
	table := []struct {
		Name      string
		RawSchema string
		Code      string
	}{
		{
			"valid login with session response",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	if *c.Name == "john" && *c.Password == "snow" {
		return &LoginWithCredentialsOuts{
			Session: &Session{ID: newString("foo")},
		}, nil
	}
	return &LoginWithCredentialsOuts{
		Error: &Error{Error: newString("invalid credentials")},
	}, nil
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, errors.New("not implemented")
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	c := Credentials{
		Name: newString("john"),
		Password: newString("snow"),
	}

	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("loginWithCredentials", c))
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var rm message
	err = json.Unmarshal(raw, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200")
	}

	if rm.Msg != "session" {
		log.Fatalf("response message was: %v", rm.Msg)
	}
	
	var sess Session
	err = json.Unmarshal(rm.Data, &sess)
	if err != nil {
		log.Fatal(err)
	}
	if *sess.ID != "foo" {
		log.Fatalf("session id was: %s", sess.ID)
	}
}
			`,
		},
		{
			"valid login with session response: http and websocket",
			fixture.TestSchemaSimpleLoginHTTPandWebsocket,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
	"strings"
	"time"
	
	"github.com/gorilla/websocket"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	if *c.Name == "john" && *c.Password == "snow" {
		return &LoginWithCredentialsOuts{
			Session: &Session{ID: newString("foo")},
		}, nil
	}
	return &LoginWithCredentialsOuts{
		Error: &Error{Error: newString("invalid credentials")},
	}, nil
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, errors.New("not implemented")
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	c := Credentials{
		Name: newString("john"),
		Password: newString("snow"),
	}

	// HTTP
	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("loginWithCredentials", c))
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var rm message
	err = json.Unmarshal(raw, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200")
	}

	if rm.Msg != "session" {
		log.Fatalf("response message was: %v", rm.Msg)
	}
	
	var sess Session
	err = json.Unmarshal(rm.Data, &sess)
	if err != nil {
		log.Fatal(err)
	}
	if *sess.ID != "foo" {
		log.Fatalf("session id was: %s", sess.ID)
	}

	// Websocket
	d := websocket.Dialer{
		Subprotocols:     []string{"p1", "p2"},
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 30 * time.Second,
	}
	url := strings.Replace(s.URL, "http://", "ws://", 1)
	conn, _, err := d.Dial(url + "/v1/websocket", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	
	m := outMessage{ Msg: "loginWithCredentials", Data: c }
	o, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	conn.WriteMessage(websocket.TextMessage, o)
	if err != nil {
		log.Fatal(err)
	}
	
	_, mr, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("read failed: %v", err)
	}
	err = json.Unmarshal(mr, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if rm.Msg != "session" {
		log.Fatalf("response message was: %v", rm.Msg)
	}
	
	err = json.Unmarshal(rm.Data, &sess)
	if err != nil {
		log.Fatal(err)
	}
	if *sess.ID != "foo" {
		log.Fatalf("session id was: %s", sess.ID)
	}
}
			`,
		},
		{
			"invalid login with error response",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	if *c.Name == "john" && *c.Password == "snow" {
		return &LoginWithCredentialsOuts{
			Session: &Session{ID: newString("foo")},
		}, nil
	}
	return &LoginWithCredentialsOuts{
		Error: &Error{Error: newString("invalid credentials")},
	}, nil
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, errors.New("not implemented")
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	c := Credentials{
		Name: newString("john"),
		Password: newString("wrong"),
	}

	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("loginWithCredentials", c))
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var rm message
	err = json.Unmarshal(raw, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200")
	}

	if rm.Msg != "error" {
		log.Fatalf("response message was: %v", rm.Msg)
	}
	
	var e Error
	err = json.Unmarshal(rm.Data, &e)
	if err != nil {
		log.Fatal(err)
	}
	if *e.Error != "invalid credentials" {
		log.Fatalf("error was: %s", e.Error)
	}
}
			`,
		},
		{
			"unknown message => 404",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"io/ioutil"
	"log"
	"io"
	"bytes"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	return nil, errors.New("not implemented")
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, errors.New("not implemented")
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	c := Credentials{
		Name: newString("john"),
		Password: newString("snow"),
	}

	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("unknownMsg", c))
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 404 {
		log.Fatal("status code not 404")
	}
}
	`,
		},
		{
			"empty response with outs and error nil => 500",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"io/ioutil"
	"log"
	"io"
	"bytes"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	return nil, nil
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, nil
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	c := Credentials{
		Name: newString("john"),
		Password: newString("snow"),
	}

	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("logout", c))
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 500 {
		log.Fatal("status code not 500")
	}
}
	`,
		},
		{
			"unparsable message => 422",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"io/ioutil"
	"log"
	"io"
	"bytes"
	"strings"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	return nil, nil
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, nil
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	res, err := http.Post(s.URL+"/v1/http", "application/json", strings.NewReader("certainly not json"))
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 422 {
		log.Fatal("status code not 422")
	}
}
	`,
		},
		{
			"empty messages",
			fixture.TestSchemaEmptyMessages,
			`
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
	"strings"
)

type Server struct{}

func(s *Server) SubscribeEmpty() error {
	return nil
}

func(s *Server) SubscribeInOnly(m *Message) error {
	return nil
}

func(s *Server) SubscribeOutsOnly() (*SubscribeOutsOnlyOuts, error) {
	return nil, nil
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	res, err := http.Post(s.URL+"/v1/http", "application/json", strings.NewReader("{ \"msg\": \"subscribeEmpty\" }"))
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200")
	}
}
	`,
		},
		{
			"in messages",
			fixture.TestSchemaEmptyMessages,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
)

type Server struct{}

func(s *Server) SubscribeEmpty() error {
	return nil
}

func(s *Server) SubscribeInOnly(m *Message) error {
	if *m.Message != "foo" {
		return errors.New("invalid in message")
	}
	return nil
}

func(s *Server) SubscribeOutsOnly() (*SubscribeOutsOnlyOuts, error) {
	return nil, nil
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()

	m := &Message{Message: newString("foo")}
	
	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("subscribeInOnly", m))
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200")
	}
}
	`,
		},
		{
			"response messages",
			fixture.TestSchemaEmptyMessages,
			`
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"bytes"
	"io/ioutil"
)

type Server struct{}

func(s *Server) SubscribeEmpty() error {
	return nil
}

func(s *Server) SubscribeInOnly(m *Message) error {
	return nil
}

func(s *Server) SubscribeOutsOnly() (*SubscribeOutsOnlyOuts, error) {
	return &SubscribeOutsOnlyOuts{ Message: &Message{newString("bar")}}, nil
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()

	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("subscribeOutsOnly", nil))
	if err != nil {
		log.Fatal(err)
	}

	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var rm message
	err = json.Unmarshal(raw, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200")
	}

	if rm.Msg != "message" {
		log.Fatalf("response message was: %v", rm.Msg)
	}
	
	var m Message
	err = json.Unmarshal(rm.Data, &m)
	if err != nil {
		log.Fatal(err)
	}
	if *m.Message != "bar" {
		log.Fatalf("message was: %s", m.Message)
	}
}
	`},
		{
			"validation fail",
			fixture.TestSchemaValidationSpec,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"bytes"
	"io/ioutil"
)

type Server struct{}

func(s *Server) SayHello(m *Message) (*SayHelloOuts, error) {
	return nil, nil
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()

	m := &Message{}
	
	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("sayHello", m))
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var rm message
	err = json.Unmarshal(raw, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 422 {
		log.Fatal("status code not 422")
	}

	if rm.Msg != "error" {
		log.Fatalf("response message was: %v", rm.Msg)
	}
	
	var e errorData
	err = json.Unmarshal(rm.Data, &e)
	if err != nil {
		log.Fatal(err)
	}
	if e.Error != "invalid message: missing message" {
		log.Fatalf("error was: %s", e.Error)
	}
}
	`,
		},
		{
			"output validation fail",
			fixture.TestSchemaValidationSpec,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"bytes"
	"io/ioutil"
)

type Server struct{}

func(s *Server) SayHello(m *Message) (*SayHelloOuts, error) {
	return &SayHelloOuts{&Message{}}, nil
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()

	m := &Message{newString("hello")}
	
	res, err := http.Post(s.URL+"/v1/http", "application/json", newMessageReader("sayHello", m))
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var rm message
	err = json.Unmarshal(raw, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 500 {
		log.Fatal("status code not 500")
	}

	if rm.Msg != "error" {
		log.Fatalf("response message was: %v", rm.Msg)
	}
	
	var e errorData
	err = json.Unmarshal(rm.Data, &e)
	if err != nil {
		log.Fatal(err)
	}
	if e.Error != "internal error" {
		log.Fatalf("error was: %s", e.Error)
	}
}
	`,
		},
	}
	for _, ts := range table {
		spec, err := jsonmsg.Parse([]byte(ts.RawSchema))
		if err != nil {
			t.Fatal(ts.Name, err)
		}
		src, err := ServerSrc(spec)
		if err != nil {
			t.Fatal(ts.Name, err)
		}

		w := &bytes.Buffer{}
		fmt.Fprintf(w, `%s`, ts.Code)
		fmt.Fprintf(w, `%s`, src)

		out, err := compileAndRun(w.Bytes())
		if err != nil {
			t.Fatal(ts.Name, err)
		}

		if out != "" {
			t.Fatalf("%v: should have produced 'ok', but produced '%v'", ts.Name, out)
		}
	}
}

func TestGenerateGoSpecHandler(t *testing.T) {
	table := []struct {
		Name      string
		RawSchema string
		Code      string
	}{
		{
			"fetchSpec message with response",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
	"strings"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	return nil, errors.New("not implemented")
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, errors.New("not implemented")
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	res, err := http.Post(s.URL+"/v1/http", "application/json", strings.NewReader("{ \"msg\": \"fetchSpec\" }"))
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	var rm message
	err = json.Unmarshal(raw, &rm)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200")
	}

	if rm.Msg != "spec" {
		log.Fatalf("response message was: %v", rm.Msg)
	}

	a := strings.Replace(string(newEmbeddedSpec()), "  ", "", -1)
	b := strings.Replace(string(rm.Data), "  ", "", -1)

	if a != b {
		log.Fatalf("response data was: \n'%s'\n should be: \n'%s'", b, a)
	}
}
			`,
		},
		{
			"GET /spec.json with response",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
	"strings"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	return nil, errors.New("not implemented")
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, errors.New("not implemented")
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	res, err := http.Get(s.URL+"/v1/spec.json")
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200, but: %v", res.StatusCode)
	}

	a := strings.Replace(string(newEmbeddedSpec()), "  ", "", -1)
	b := strings.Replace(string(raw), "  ", "", -1)
	a = strings.Replace(a, "\n", "", -1)
	b = strings.Replace(b, "\n", "", -1)

	if a != b {
		log.Fatalf("response data was: \n'%s'\n should be: \n'%s'", b, a)
	}
}
			`,
		},
		{
			"GET /spec with response",
			fixture.TestSchemaSimpleLogin,
			`
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"log"
	"io"
	"io/ioutil"
	"bytes"
)

type Server struct{}

func(s *Server) LoginWithCredentials(c *Credentials) (*LoginWithCredentialsOuts, error) {
	return nil, errors.New("not implemented")
}

func(s *Server) Logout(sess *Session) (*LogoutOuts, error) {
	return nil, errors.New("not implemented")
}

func main() {
	s := httptest.NewServer(NewAPIMux(&Server{}))
	defer s.Close()
	
	res, err := http.Get(s.URL+"/v1/spec")
	if err != nil {
		log.Fatal(err)
	}
	raw, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("status code not 200, but: %v", res.StatusCode)
	}
	
	if res.Header.Get("Content-Type") != "text/html" {
		log.Fatal("content type not text/html, but: %v", res.Header.Get("Content-Type"))
	}

	if len(raw) < 400 {
		log.Fatalf("Response content suspiciously short: %v", raw)
	}
}
			`,
		},
	}
	for _, ts := range table {
		spec, err := jsonmsg.Parse([]byte(ts.RawSchema))
		if err != nil {
			t.Fatal(ts.Name, err)
		}
		src, err := ServerSrc(spec)
		if err != nil {
			t.Fatal(ts.Name, err)
		}

		w := &bytes.Buffer{}
		fmt.Fprintf(w, `%s`, ts.Code)
		fmt.Fprintf(w, `%s`, src)

		out, err := compileAndRun(w.Bytes())
		if err != nil {
			t.Fatal(ts.Name, err)
		}

		if out != "" {
			t.Fatalf("%v: should have produced 'ok', but produced '%v'", ts.Name, out)
		}
	}
}

// compiles the given code, runs it and returns the response
func compileAndRun(code []byte) (string, error) {
	const name = "tmp"
	os.RemoveAll(name)
	err := os.Mkdir(name, 0700)
	if err != nil {
		return "", err
	}

	// write src
	err = ioutil.WriteFile(name+"/main.go", code, 0700)
	if err != nil {
		return "", err
	}

	// get deps
	cmd := exec.Command("bash", "-c", "go get .")
	cmd.Dir = name
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}

	// compile
	cmd = exec.Command("bash", "-c", "go build")
	cmd.Dir = name
	out, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}

	// execute
	cmd = exec.Command("bash", "-c", "./"+name)
	cmd.Dir = name
	out, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}

	os.RemoveAll(name)
	return string(out), nil
}
