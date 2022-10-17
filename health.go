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
	mutex   sync.RWMutex
	reports map[string][]error
}

type Response struct {
	Status int     `json:"status"`
	Errors []error `json:"errors,omitempty"`
}

func NewHealth() *Health {
	return &Health{
		reports: map[string][]error{},
		mutex:   sync.RWMutex{},
	}
}

func (health *Health) Alert(err error, keys ...string) {
	key := strings.Join(keys, "@")

	health.mutex.Lock()
	if _, ok := health.reports[key]; !ok {
		health.reports[key] = []error{err}
	} else {
		health.reports[key] = append(health.reports[key], err)
	}
	health.mutex.Unlock()
}

func (health *Health) Resolve(keys ...string) {
	key := strings.Join(keys, "@")

	health.mutex.Lock()
	delete(health.reports, key)
	health.mutex.Unlock()
}

func (health *Health) GetStatus() int {
	health.mutex.RLock()
	size := len(health.reports)
	health.mutex.RUnlock()

	if size == 0 {
		return 0
	}

	return 1
}

func (health *Health) GetErrors() []error {
	health.mutex.RLock()
	defer health.mutex.RUnlock()

	errors := make([]error, 0, len(health.reports))
	for _, errs := range health.reports {
		for _, err := range errs {
			errors = append(errors, Error(health.formatError(err)))
		}
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
	health.mutex.Lock()
	defer health.mutex.Unlock()

	return len(health.reports) > 0
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
	health.mutex.RLock()
	defer health.mutex.RUnlock()

	if !health.HasErrors() {
		return Response{
			Status: health.GetStatus(),
		}
	}

	errors := make([]error, 0, len(health.reports))
	for _, errs := range health.reports {
		for _, err := range errs {
			if _, ok := err.(karma.Karma); ok {
				continue
			}

			errors = append(errors, Error(err.Error()))
		}
	}

	return Response{
		Status: health.GetStatus(),
		Errors: errors,
	}
}
