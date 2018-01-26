package jsonmsg

import (
	"encoding/json"
	"fmt"
)

// Parse a schema into an Index of Schemas
func ExampleParse() {
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

	// parse into index
	spec, err := Parse([]byte(schema))
	if err != nil {
		panic(err)
	}

	fmt.Printf("http endpoint: %s\n", spec.Endpoints.URLs["http"])
	fmt.Printf("websocket endpoint: %s\n", spec.Endpoints.URLs["websocket"])
	fmt.Printf("base url: %s\n", spec.Endpoints.BaseURL)
	fmt.Printf("base path: %s\n", spec.Endpoints.BasePath)

	fmt.Printf("findUser name: %s\n", spec.Messages["findUser"].Name)
	fmt.Printf("findUser in name: %s\n", spec.Messages["findUser"].InSchema.Name)
	fmt.Printf("findUser in group: %s\n", spec.GroupedMessages["user"]["findUser"].InSchema.Name)

	// Output:
	// http endpoint: https://jsonmsg.github.io/v1/http
	// websocket endpoint: wss://jsonmsg.github.io/v1/websocket
	// base url: https://jsonmsg.github.io/v1
	// base path: /v1
	// findUser name: FindUser
	// findUser in name: UserQuery
	// findUser in group: UserQuery
}

// Generate a sample message conforming to the specified message schema
func ExampleMessage_NewInstance() {
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

	// parse into spec
	spec, err := Parse([]byte(schema))
	if err != nil {
		panic(err)
	}

	// create go instance
	inst, err := spec.Messages["findUser"].NewInstance()
	if err != nil {
		panic(err)
	}

	// marshal to json
	raw, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", raw)
	// Output:
	// {
	//   "data": {
	//     "id": "string"
	//   },
	//   "msg": "findUser"
	// }
}
