package fixture

import (
	"encoding/json"
	"testing"
)

func TestFixtureUnmarshal(t *testing.T) {
	fs := map[string]string{
		"TestSchemaSimpleLogin":                 TestSchemaSimpleLogin,
		"TestSchemaSimpleLoginHTTPandWebsocket": TestSchemaSimpleLoginHTTPandWebsocket,
		"TestSchemaEmptyMessages":               TestSchemaEmptyMessages,
		"TestSchemaValidationSpec":              TestSchemaValidationSpec,
	}
	for k, v := range fs {
		var o interface{}
		err := json.Unmarshal([]byte(v), &o)
		if err != nil {
			t.Fatalf("fixture %s does not unmarshal: %s", k, err)
		}
	}
}
