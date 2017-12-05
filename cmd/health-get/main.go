package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/bclicn/color"
	"github.com/docopt/docopt-go"
	health "github.com/reconquest/health-go"
	karma "github.com/reconquest/karma-go"
	"github.com/reconquest/regexputil-go"
)

var (
	version = "[manual build]"
	usage   = "health-get " + version + `

Retrieves health status.

Usage:
  health-get [options] <url>
  health-get -h | --help
  health-get --version

Options:
  -h --help  Show this screen.
  --version  Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	url := args["<url>"].(string)

	resource, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	var response health.Response
	err = json.NewDecoder(resource.Body).Decode(&response)
	if err != nil {
		log.Fatal(err)
	}

	var tree *karma.Context
	if response.Status == 0 {
		tree = tree.Describe("status", color.BGreen("ok"))
	} else {
		tree = tree.Describe("status", color.BRed("error"))

		var root karma.Reason
		root = karma.Push(
			fmt.Sprintf("%d", len(response.Errors)),
			getReasons(response.Errors)...,
		)

		tree = tree.Describe("errors", root)
	}

	fmt.Println(tree.Reason(url))
}

func getReasons(rows []string) []karma.Reason {
	matcher := regexp.MustCompile(`(?P<keyvalue>\[(?P<key>[^=]+)=(?P<value>[^\]]+)\])`)

	var reasons []karma.Reason
	for _, row := range rows {
		var context *karma.Context
		var message = row

		matches := matcher.FindAllStringSubmatch(row, -1)
		for _, submatches := range matches {
			var (
				key      = regexputil.Subexp(matcher, submatches, "key")
				value    = regexputil.Subexp(matcher, submatches, "value")
				keyvalue = regexputil.Subexp(matcher, submatches, "keyvalue")
			)

			context = context.Describe(key, value)
			message = strings.Replace(message, keyvalue, "", -1)
		}

		message = strings.TrimSpace(message)

		reasons = append(reasons, context.Reason(message))
	}

	return reasons
}
