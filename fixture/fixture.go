// Package fixture provides common schemas for testing and evaluation
package fixture

const (
	// A simple login/logout schema http enpoint only
	TestSchemaSimpleLogin = `
{
	"endpoints": {
		"http": "http://api.specc.io/v1"
	},
	"messages": {
		"loginWithCredentials": {
			"in": "#/definitions/credentials",
			"outs": [
				"#/definitions/session",
				"#/definitions/error"
			],
			"group": "login"
		},
		"logout": {
			"in": "#/definitions/session",
			"outs": [
				"#/definitions/message"
			],
			"group": "logout"
		}
	},
	"definitions": {
		"credentials": {
			"type": "object",
			"properties": {
				"name": {
					"type": "string"
				},
				"password": {
					"type": "string"
				}
			}
		},
		"session": {
			"type": "object",
			"properties": {
				"id": {
					"type": "string"
				}
			}
		},
		"error": {
			"type": "object",
			"properties": {
				"error": {
					"type": "string"
				}
			}
		},
		"message": {
			"type": "object",
			"properties": {
				"message": {
					"type": "string"
				}
			}
		}
	}
}
`
	// A simple login/logout schema http and websocket enpoint
	TestSchemaSimpleLoginHTTPandWebsocket = `
{
	"endpoints": {
		"http": "https://api.specc.io/v1",
		"websocket": "wss://api.specc.io/v1"
	},
	"messages": {
		"loginWithCredentials": {
			"in": "#/definitions/credentials",
			"outs": [
				"#/definitions/session",
				"#/definitions/error"
			],
			"group": "login"
		},
		"logout": {
			"in": "#/definitions/session",
			"outs": [
				"#/definitions/message"
			],
			"group": "logout"
		}
	},
	"definitions": {
		"credentials": {
			"type": "object",
			"properties": {
				"name": {
					"type": "string"
				},
				"password": {
					"type": "string"
				}
			}
		},
		"session": {
			"type": "object",
			"properties": {
				"id": {
					"type": "string"
				}
			}
		},
		"error": {
			"type": "object",
			"properties": {
				"error": {
					"type": "string"
				}
			}
		},
		"message": {
			"type": "object",
			"properties": {
				"message": {
					"type": "string"
				}
			}
		}
	}
}
`
	// A schema with empty message inputs and message outputs
	TestSchemaEmptyMessages = `
{
	"endpoints": {
		"http": "http://api.specc.io/v1"
	},
	"messages": {
		"subscribeEmpty": {
		},
		"subscribeInOnly": {
			"in": "#/definitions/message"
		},
		"subscribeOutsOnly": {
			"outs": [
				"#/definitions/message"
			]
		}
	},
	"definitions": {
		"message": {
			"type": "object",
			"properties": {
				"message": {
					"type": "string"
				}
			}
		}
	}
}
`
	// A schema with validations
	TestSchemaValidationSpec = `
{
	"endpoints": {
		"http": "http://api.specc.io/v1"
	},
	"messages": {
		"sayHello": {
			"in": "#/definitions/message",
			"outs": [
				"#/definitions/message"
			]
		}
	},
	"definitions": {
		"message": {
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
)
