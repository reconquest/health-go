package health

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/reconquest/karma-go"
)

const (
	HierarchyDelimiterUnicode = " → "
	HierarchyDelimiterASCII   = ": "
)

var (
	DefaultHierarchyDelimiter = HierarchyDelimiterUnicode
)

type Error string

func (err Error) Error() string {
	return string(err)
}

func (err Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(err))
}

type Health struct {
	*sync.Mutex
	keys   []string
	errors []error
}

type Response struct {
	Status int     `json:"status"`
	Errors []error `json:"errors,omitempty"`
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
			health.errors[index] = err

			return
		}
	}

	health.keys = append(health.keys, key)
	health.errors = append(health.errors, err)
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

func (health *Health) GetStatus() int {
	health.Lock()
	defer health.Unlock()

	if len(health.errors) > 0 {
		return 1
	}

	return 0
}

func (health *Health) GetErrors() []error {
	health.Lock()
	defer health.Unlock()

	errors := []error{}
	for _, err := range health.errors {
		errors = append(errors, Error(health.formatError(err)))
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
			message += err.Message +
				DefaultHierarchyDelimiter + health.formatError(err.Reason)
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
	return json.Marshal(health.GetResponse())
}

func (health *Health) GetResponse() Response {
	if !health.HasErrors() {
		return Response{
			Status: health.GetStatus(),
		}
	}

	return Response{
		Status: health.GetStatus(),
		Errors: health.GetErrors(),
	}
}

func (health *Health) GetExpandedResponse() Response {
	if !health.HasErrors() {
		return Response{
			Status: health.GetStatus(),
		}
	}

	for i, err := range health.errors {
		if _, ok := err.(karma.Karma); ok {
			continue
		}

		health.errors[i] = Error(err.Error())
	}

	return Response{
		Status: health.GetStatus(),
		Errors: health.errors,
	}
}
