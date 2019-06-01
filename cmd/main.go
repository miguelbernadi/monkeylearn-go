package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
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
	if err != nil {
		log.Panic(err)
	}
	defer jsonFile.Close()
	fmt.Printf("Reading from %s\n", *filename)

	docs := load(jsonFile)
	fmt.Printf("Documents to classify: %d\n", len(docs))

	rate := time.Minute / time.Duration(*rpm)

	// Initialize the API client
	client := monkeylearn.
		NewDefaultClient(*token).
		SetMaxRate(rate).
		SetClassificationModel(*classifier)

	fmt.Printf("Batch size: %d\n", *batchSize)
	fmt.Printf("Number of batches: %d\n", len(docs) / *batchSize + 1)

	// Start background request processor
	go client.RunProcessor()

	// Start processing documents
	for _, doc := range docs {
		do := monkeylearn.DataObject{Text: doc, ExternalID: nil}
		client.ProcessDocuments(do)
	}
	client.Batch(*batchSize)

	// We can now read results
	for resp := range client.Results() {
		log.Printf("%#v\n", resp)
	}
	fmt.Printf("Remaining credits: %d / %d\n", client.Limits.RequestRemaining, client.Limits.RequestLimit)
}

func load(stream io.Reader) []string {
	data, err := ioutil.ReadAll(stream)
	if err != nil {
		log.Panic(err)
	}

	var docs []string
	err = json.Unmarshal(data, &docs)
	if err != nil {
		log.Panic(err)
	}

	return docs
}
