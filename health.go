package health

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/reconquest/karma-go"
)

type Health struct {
	*sync.Mutex
	errors map[string]string
}

type response struct {
	Status string   `json:"status"`
	Errors []string `json:"errors,omitempty"`
}

func NewHealth() *Health {
	return &Health{
		Mutex:  &sync.Mutex{},
		errors: map[string]string{},
	}
}

func (health *Health) Alert(err error, keys ...string) {
	health.Lock()
	defer health.Unlock()

	health.errors[strings.Join(keys, "@")] = health.formatError(err)
}

func (health *Health) Resolve(keys ...string) {
	health.Lock()
	defer health.Unlock()

	delete(health.errors, strings.Join(keys, "@"))
}

func (health *Health) GetErrors() []string {
	health.Lock()
	defer health.Unlock()

	errors := []string{}
	for _, err := range health.errors {
		errors = append(errors, err)
	}

	return errors
}

func (health *Health) formatError(reason interface{}) string {
	err, ok := reason.(karma.Karma)
	if !ok {
		return fmt.Sprint(reason)
	}

	var message string
	switch value := err.Reason.(type) {
	case nil:
		message += err.Message

	case []karma.Reason:
		reasons := []string{}
		for _, reason := range value {
			reasons = append(reasons, health.formatError(reason))
		}

		message += strings.Join(reasons, "; ")

	default:
		message += err.Message + ": " + health.formatError(err.Reason)
	}

	var pairs []string
	err.Context.Walk(func(key string, value interface{}) {
		pairs = append(pairs, fmt.Sprintf("[%s=%v]", key, value))
	})
	if len(pairs) > 0 {
		message += " " + strings.Join(pairs, " ")
	}

	return message
}

func (health *Health) MarshalJSON() ([]byte, error) {
	errors := health.GetErrors()
	if len(errors) == 0 {
		return json.Marshal(response{Status: "ok"})
	}

	return json.Marshal(response{Status: "error", Errors: errors})
}
