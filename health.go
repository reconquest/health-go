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
	keys   []string
	errors []string
}

type response struct {
	Status string   `json:"status"`
	Errors []string `json:"errors,omitempty"`
}

func NewHealth() *Health {
	return &Health{
		Mutex: &sync.Mutex{},
	}
}

func (health *Health) Alert(err error, keys ...string) {
	health.Lock()
	defer health.Unlock()

	key := strings.Join(keys, "@")

	for index, stored := range health.keys {
		if stored == key {
			health.errors[index] = health.formatError(err)

			return
		}
	}

	health.keys = append(health.keys, key)
	health.errors = append(health.errors, health.formatError(err))
}

func (health *Health) Resolve(keys ...string) {
	health.Lock()
	defer health.Unlock()

	key := strings.Join(keys, "@")

	for index, stored := range health.keys {
		if stored == key {
			health.keys = append(
				health.keys[:index],
				health.keys[index+1:]...,
			)
			health.errors = append(
				health.errors[:index],
				health.errors[index+1:]...,
			)

			return
		}
	}
}

func (health *Health) GetStatus() string {
	health.Lock()
	defer health.Unlock()

	if len(health.errors) > 0 {
		return "error"
	}

	return "ok"
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
		if err.Message != "" {
			message += err.Message + ": " + health.formatError(err.Reason)
		} else {
			message += health.formatError(err.Reason)
		}
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

func (health *Health) HasErrors() bool {
	health.Lock()
	defer health.Unlock()

	return len(health.errors) > 0
}

func (health *Health) MarshalJSON() ([]byte, error) {
	if !health.HasErrors() {
		return json.Marshal(response{Status: "ok"})
	}

	return json.Marshal(
		response{
			Status: health.GetStatus(),
			Errors: health.GetErrors(),
		},
	)
}
