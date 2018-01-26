/*
Package golang generates go sources implementing a jsonmsg.Spec.

Server

The generated sources for a server will include all types with validations, an API interface with all messages and a NewAPIMux function expecting a struct implementing the API interface.
The generated mux will then automatically validate and dispatch incoming messages to interface functions.

First, define and parse a spec into a go file:

	schema := `
	{
	  "endpoints": {
	    "host": "jsonmsg.github.io/v1",
	    "protocols": ["http", "websocket"],
	    "tls": true
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
	spc, err := jsonmsg.Parse(spec)
	if err != nil {
		panic(err)
	}

	// generate server source for a package
	src, err := ServerPackageSrc(spc, "main")
	if err != nil {
		panic(err)
	}

	// write to file
	err = ioutil.WriteFile("api.gen.go", src, 0644)
	if err != nil {
		panic(err)
	}

The api.gen.go file now contains all types, the API interface and NewAPIMux function:

	type API interface {
		FindUser(*UserQuery) (*FindUserOuts, error)
	}

	func NewAPIMux(i API) *http.ServeMux {
		...
	}

	type Error struct {
		Message *string `json:"message,omitempty"`
	}

	type User struct {
		ID   *string `json:"id,omitempty"`
		Name *string `json:"name,omitempty"`
	}

	type UserQuery struct {
		ID *string `json:"id,omitempty"`
	}

	func (t *Error) Validate() error {
		if t.Message == nil {
			return errors.New("invalid error: missing message")
		}

		return nil
	}

	func (t *User) Validate() error {
		...
	}

To run a server with the API you need to implement the API interface, e.g. in main.go:

	package main

	import (
		"fmt"
		"log"
		"net/http"
	)

	type Server struct{}

	func (s *Server) FindUser(q *UserQuery) (*FindUserOuts, error) {
		// search user
		return &FindUserOuts{
			User: &User{
				ID:   newString("visurgif"),
				Name: newString("Foo Bar"),
			},
		}, nil
	}

	func main() {
		h := NewAPIMux(&Server{})

		fmt.Println("Running http://localhost:8000")
		log.Fatal(http.ListenAndServe("localhost:8000", h))
	}

When updating the schema.json, api.gen.go is overriden with the latest interface definitions.
The implemented server then needs to be adapted to conform to the latest interface.
Testing of the server implementation does not involve any HTTP/websocket stack.

Client

Client is not implemented yet.
*/
package golang
