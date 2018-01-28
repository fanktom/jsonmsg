package jsonmsg

import (
	"testing"

	"github.com/tfkhsr/jsonmsg/fixture"
)

func TestSimpleLogin(t *testing.T) {
	spc, err := Parse([]byte(fixture.TestSchemaSimpleLogin))
	if err != nil {
		t.Fatal(err)
	}
	if spc.Endpoints["http"].String() != "http://api.specc.io/v1/http" {
		t.Fatalf("invalid http url: %v", spc.Endpoints["http"].String())
	}

	m, ok := spc.Messages["loginWithCredentials"]
	if !ok {
		t.Fatalf("message loginWithCredentials was not present")
	}
	if m.In != "#/definitions/credentials" {
		t.Fatalf("invalid message in")
	}
	if m.Outs[0] != "#/definitions/session" {
		t.Fatalf("invalid message out session")
	}
	if m.Outs[1] != "#/definitions/error" {
		t.Fatalf("invalid message out error")
	}

	if _, ok := spc.GroupedMessages["login"]["loginWithCredentials"]; !ok {
		t.Fatalf("loginWithCredentials was not in login group: %v", spc.GroupedMessages)
	}

	if _, ok := spc.GroupedMessages["logout"]["logout"]; !ok {
		t.Fatalf("logout was not in logout group: %v", spc.GroupedMessages)
	}

	tbl := []string{
		"#/definitions/credentials",
		"#/definitions/session",
		"#/definitions/error",
	}
	for _, d := range tbl {
		def := spc.Definitions[d]
		if def == nil {
			t.Fatalf("missing schema: %v", d)
		}
	}
}

func TestSimpleLoginWithHTTPAndWebsockets(t *testing.T) {
	spc, err := Parse([]byte(fixture.TestSchemaSimpleLoginHTTPandWebsocket))
	if err != nil {
		t.Fatal(err)
	}
	if spc.Endpoints["http"].String() != "https://api.specc.io/v1/http" {
		t.Fatalf("invalid http url: %v", spc.Endpoints["http"].String())
	}
	if spc.Endpoints["websocket"].String() != "wss://api.specc.io/v1/websocket" {
		t.Fatalf("invalid websocket url: %v", spc.Endpoints["websocket"].String())
	}
}

func TestJSONSpec(t *testing.T) {
	spc, err := Parse([]byte(fixture.TestSchemaSimpleLogin))
	if err != nil {
		t.Fatal(err)
	}

	out, err := spc.JSONSpec()
	if err != nil {
		t.Fatal(err)
	}

	spc2, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if spc.Endpoints["http"].String() != spc2.Endpoints["http"].String() {
		t.Fatal("different urls")
	}
}
