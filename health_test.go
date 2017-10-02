package health

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	untabber = strings.NewReplacer("\n\t\t", "\n")
)

func TestHealth_Alert_SavesError(t *testing.T) {
	test := assert.New(t)

	health := NewHealth()
	health.Alert(errors.New("a"), "key1", "key2")

	test.EqualValues([]string{"a"}, health.GetErrors())
}

func TestHealth_Alert_UsesKeys(t *testing.T) {
	test := assert.New(t)

	health := NewHealth()
	health.Alert(errors.New("a"), "key1", "key2")

	test.EqualValues(map[string]string{"key1@key2": "a"}, health.errors)
}

func TestHealth_Resolve_RemovesError(t *testing.T) {
	test := assert.New(t)

	health := NewHealth()
	health.Alert(errors.New("a"), "key1", "key2")
	health.Alert(errors.New("b"), "key3", "key4")

	health.Resolve("key3", "key4")

	test.EqualValues([]string{"a"}, health.GetErrors())
	test.EqualValues(map[string]string{"key1@key2": "a"}, health.errors)
}

func TestHealth_MarshalJSON_ReturnsOkForNoErrors(t *testing.T) {
	test := assert.New(t)

	health := NewHealth()

	buffer := bytes.NewBuffer(nil)

	encoder := json.NewEncoder(buffer)
	encoder.SetIndent("", "\t")

	err := encoder.Encode(health)
	test.NoError(err)

	test.Equal(
		untabber.Replace(`{
			"status": "ok"
		}
		`),
		string(buffer.Bytes()),
	)
}
