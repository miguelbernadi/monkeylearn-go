package monkeylearn

import (
	"encoding/json"
	"fmt"
	"log"
)

// DataObject is used to serialize Messages to MonkeyLearn classifiers
type DataObject struct {
	Text string `json:"text"`
	ExternalID *string `json:"external_id"`
}

// Batch represents a group of DataObjects for processing together in
// a single request
type Batch struct {
	Data []DataObject `json:"data"`
}

// NewBatch returns an empty Batch
func NewBatch() *Batch {
	return &Batch{}
}

// Add adds a document to an existing Batch, updates the referenced
// document and returns it.
func (b *Batch) Add(document DataObject) *Batch {
	b.Data = append(b.Data, document)
	return b
}

// Classify runs the classifier against the specified model
func (b Batch) Classify(model string, client *Client) ([]Result, error) {
	defer startTimer(model)()

	data, err := json.Marshal(b)
	if err != nil { log.Panic(err) }

	return client.Process(fmt.Sprintf(classifierURL, model), data)
}

// Extract runs the extractor against the specified model
func (b Batch) Extract(model string, client *Client) ([]Result, error) {
	defer startTimer(model)()

	data, err := json.Marshal(b)
	if err != nil { log.Panic(err) }

	return client.Process(fmt.Sprintf(extractorURL, model), data)
}

// SplitInBatches takes a list of documents and the expected size of
// each Batch and returns a list of Batches with batchSize elements
// each.
func SplitInBatches(docs []DataObject, batchSize int) []*Batch {
	defer startTimer("Split in batches")()
	batches := []*Batch{}
	count := 0
	var tmpbatch *Batch
	for _, doc := range docs {
		if count % batchSize == 0 {
			tmpbatch = NewBatch()
		}
		count++
		tmpbatch.Add(doc)
		if count % batchSize == 0 || count == len(docs) {
			batches = append(batches, tmpbatch)
		}
	}
	return batches
}

