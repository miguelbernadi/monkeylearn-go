package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/miguelbernadi/monkeylearn-go/pkg/monkeylearn"
)

func main() {
	token := flag.String("token", "", "Monkeylearn token")
	classifier := flag.String("classifier", "", "Monkeylearn classifier ID")
	flag.Parse()

	if *token == "" {
		log.Fatal("Token is mandatory")
	}
	client := monkeylearn.NewClient(*token)

	docs := []string{
		"aabb",
		"bbaa",
	}

	for resp := range loop(time.Minute / 120, docs, client, *classifier) {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil { log.Panic(err) }
		log.Printf("%#v\n", string(body))
	}
}

func loop(rate time.Duration, docs []string, client *monkeylearn.Client, classifier string) (out chan *http.Response) {
	out = make(chan *http.Response)

	throttle := time.Tick(rate)
	var wg sync.WaitGroup
	for _, doc := range docs {
		wg.Add(1)
		<-throttle  // rate limit
		go func(doc string) {
			data := &monkeylearn.Documents{
				[]monkeylearn.DataObject{
					{ Text: doc },
				},
			}
			out <- client.Classify(classifier, data)
			wg.Done()
		}(doc)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
