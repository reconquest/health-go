package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

type Handler struct {
	health *Health
}

func (handler *Handler) ServeHTTP(response http.ResponseWriter, _ *http.Request) {
	encoder := json.NewEncoder(response)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(handler.health)
	if err != nil {
		panic(err)
	}
}

func Example_Health_DefaultResponse() {
	health := NewHealth()

	server := httptest.NewServer(&Handler{health})

	defer server.Close()

	response, err := http.Get(server.URL)
	if err != nil {
		panic(err)
	}

	fmt.Println(body(response))

	// Output:
	// {
	//   "status": 0
	// }
}

func Example_Health_AlertErrors() {
	health := NewHealth()

	health.Alert(
		errors.New("no meaning of life"),
		"real", "talk",
	)

	health.Alert(
		errors.New("time is a flat circle"),
		"true", "detective",
	)

	server := httptest.NewServer(&Handler{health})
	defer server.Close()

	response, err := http.Get(server.URL)
	if err != nil {
		panic(err)
	}

	fmt.Println(body(response))

	// Output:
	// {
	//   "status": 1,
	//   "errors": [
	//     "no meaning of life",
	//     "time is a flat circle"
	//   ]
	// }
}

func Example_Health_ResovleErrors() {
	health := NewHealth()

	health.Alert(
		errors.New("everything is the worst"),
		"life",
		"cycle",
	)

	server := httptest.NewServer(&Handler{health})
	defer server.Close()

	health.Resolve("life", "cycle")

	response, err := http.Get(server.URL)
	if err != nil {
		panic(err)
	}

	fmt.Println(body(response))

	// Output:
	// {
	//   "status": 0
	// }
}

func body(response *http.Response) string {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}
