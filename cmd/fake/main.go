package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var counts, errorRate, errors int
var errorCodes []int

func replyOK() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		message := strings.TrimPrefix("/", r.URL.Path)
		message = message + " served ok"

		responseCode := http.StatusOK
		w.WriteHeader(responseCode)
		w.Write([]byte(message))
	})
}

func countRequests(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		message := strings.TrimPrefix("/", r.URL.Path)
		counts = counts + 1
		// If errorRate is 0, no errors should be thrown;
		// else, every errorRate requests will raise an error.
		if errorRate > 0 && counts%errorRate == 0 {
			errors = errors + 1
			http.Error(
				w,
				"Errored request to "+message,
				errorCodes[errors%len(errorCodes)],
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func printCounts() {
	for range time.Tick(10 * time.Second) {
		fmt.Printf("requests=%v errors=%v\n", counts, errors)
		counts = 0
		errors = 0
	}
}

func parseInts(ints *string) (intList []int) {
	for _, n := range strings.Split(*ints, ",") {
		value, err := strconv.Atoi(n)
		if err != nil {
			panic(err)
		}
		intList = append(intList, value)
	}
	return intList
}

func main() {
	flag.IntVar(&errorRate, "errors", 0, "An error response will be sent every this number of requests.")
	errorCodesList := flag.String("error-codes", "500", "comma-separated list of error codes to use when replying an error.")
	flag.Parse()

	errorCodes = parseInts(errorCodesList)
	counts = 0
	errors = 0

	go printCounts()

	http.HandleFunc("/", countRequests(replyOK()))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
