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

	// parse into index
	spec, err := Parse([]byte(schema))
	if err != nil {
		panic(err)
	}

	fmt.Printf("findUser name: %s\n", spec.Messages["findUser"].Name)
	fmt.Printf("findUser in name: %s\n", spec.Messages["findUser"].InSchema.Name)
	fmt.Printf("findUser in group: %s\n", spec.GroupedMessages["user"]["findUser"].InSchema.Name)

	// Output:
	// findUser name: FindUser
	// findUser in name: UserQuery
	// findUser in group: UserQuery
}

// Generate a sample message conforming to the specified message schema
func ExampleMessage_NewInstance() {
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
