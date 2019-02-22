package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/miguelbernadi/monkeylearn-go/pkg/monkeylearn"
)

func main() {
	token := flag.String("token", "", "Monkeylearn token")
	classifier := flag.String("classifier", "", "Monkeylearn classifier ID")
	rpm := flag.Int64("rpm", 120, "Requests per minute (should be lower than API rate limit)")
	batchSize := flag.Int("batch", 1, "Documents per batch")
	filename := flag.String("file", "data.json", "File containing the documents to process")
	flag.Parse()

	if *token == "" {
		log.Fatal("Token is mandatory")
	}

	jsonFile, err := os.Open(*filename)
	if err != nil { log.Panic(err) }
	defer jsonFile.Close()
	fmt.Printf("Reading from %s\n", *filename)

	docs := load(jsonFile)
	fmt.Printf("Documents to classify: %d\n", len(docs))

	var dataObjects = []monkeylearn.DataObject{}
	for _, doc := range docs {
		dataObjects = append(dataObjects, monkeylearn.DataObject{Text: doc, ExternalID: nil})
	}
	batches := monkeylearn.SplitInBatches(dataObjects, *batchSize)
	fmt.Printf("Batch size: %d\n", *batchSize)
	fmt.Printf("Number of batches: %d\n", len(batches))

	client := monkeylearn.NewClient(*token)
	for resp := range loop(time.Minute / time.Duration(*rpm), batches, client, *classifier) {
		log.Printf("%#v\n", resp)
	}
	fmt.Printf("Remaining credits: %d / %d\n", client.RequestRemaining, client.RequestLimit)
}

func loop(rate time.Duration, batches []*monkeylearn.Batch, client *monkeylearn.Client, classifier string) (out chan monkeylearn.Result) {
	out = make(chan monkeylearn.Result)

	throttle := time.Tick(rate)
	var wg sync.WaitGroup
	for _, batch := range batches {
		wg.Add(1)
		<-throttle  // rate limit
		go func(batch monkeylearn.Batch) {
			resp, err := batch.Classify(classifier, client)
			if err != nil { log.Panic(err) }
			for _, doc := range resp {
				out <- doc
			}
			wg.Done()
		}(*batch)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func load(stream io.Reader) []string {
	data, err := ioutil.ReadAll(stream)
	if err != nil { log.Panic(err) }

	var docs []string
	err = json.Unmarshal(data, &docs)
	if err != nil { log.Panic(err) }

	return docs
}
