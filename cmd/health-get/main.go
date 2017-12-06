package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
  -h --help                Show this screen.
  -d --delimiter <string>  Hierarchy delimiter.
                            [default: → ]
  -v --verbose             Be verbose.
  --version                Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	var (
		target    = args["<url>"].(string)
		delimiter = args["--delimiter"].(string)
		verbose   = args["--verbose"].(bool)
	)

	uri, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}

	fixed := false
	if uri.Scheme == "" {
		fixed = true
		uri.Scheme = "http"

		// workaround for https://play.golang.org/p/EjNV-iDLZv
		newtarget := uri.String()

		uri, err = url.Parse(newtarget)
		if err != nil {
			log.Fatalf("%s: %s", newtarget, err)
		}
	}

	if uri.Path == "" {
		fixed = true
		uri.Path = "/health"
	}

	if verbose && fixed {
		log.Printf("URL fixed to %s", uri.String())
	}

	target = uri.String()

	resource, err := http.Get(target)
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
			getReasons(response.Errors, delimiter)...,
		)

		tree = tree.Describe("errors", root)
	}

	fmt.Println(tree.Reason(target))
}

func getReasons(rows []string, delimiter string) []karma.Reason {
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

		var reason karma.Reason

		parts := strings.Split(message, delimiter)
		for i := len(parts) - 1; i >= 0; i-- {
			trimed := strings.TrimSpace(parts[i])
			if i == len(parts)-1 {
				reason = context.Reason(trimed)
				continue
			}

			reason = karma.Karma{
				Message: trimed,
				Reason:  reason,
			}
		}

		reasons = append(reasons, reason)
	}

	return reasons
}
