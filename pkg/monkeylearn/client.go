package monkeylearn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
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
func (b *Batch) Add(document string) *Batch {
	b.Data = append(b.Data, DataObject{Text: document})
	return b
}

// SplitInBatches takes a list of documents and the expected size of
// each Batch and returns a list of Batches with batchSize elements
// each.
func SplitInBatches(docs []string, batchSize int) []*Batch {
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

// Client holds the authentication data to connect to the MonkeyLearn
// API and is used as gateway to operate with the API
type Client struct {
	token string
	client *http.Client
	RequestLimit, RequestRemaining int
}

// NewClient returns a new Client initialized with an authentication token
func NewClient(token string) *Client {
	return &Client{token: token, client: &http.Client{} }
}

// Rate limiting
// {
// 	"status_code": 429,
// 	"error_code": "CONCURRENCY_RATE_LIMIT",
// 	"detail": "Request was throttled. Too many concurrent requests."
// }

// Classify takes an identifier for a model, a Batch to process and
// returns the a ClassifyResult list for all documents, or an error.
func (c *Client) Classify(model string, docs Batch) ([]ClassifyResult, error) {
	defer startTimer(model)()

	url := "https://api.monkeylearn.com/v3"
	endpoint := fmt.Sprintf("%s/classifiers/%s/classify/", url, model)
	data, err := json.Marshal(docs)
	if err != nil { log.Panic(err) }

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil { log.Panic(err) }

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", c.token))
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil { log.Panic(err) }

	// We get rate limited. Do something
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("Request got ratelimited. Model: %s", model)
	}

	// Not succesful? Better error out
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Unsuccessful request: %s", resp.Status)
	}

	// Only if request is successful
	c.RequestRemaining, err = strconv.Atoi(resp.Header.Get("X-Query-Limit-Remaining"))
	if err != nil { log.Panic(err) }
	c. RequestLimit, err = strconv.Atoi(resp.Header.Get("X-Query-Limit-Limit"))
	if err != nil { log.Panic(err) }

	// Deserialize response and deal with it
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { log.Panic(err) }

	var res []ClassifyResult
	err = json.Unmarshal(body, &res)
	if err != nil { log.Panic(err) }

	return res, nil
}

// ClassifyResult holds the results of classifying a document
type ClassifyResult struct {
	Text string
	ExternalID int `json:"external_id"`
	Error bool
	ErrorDetail string `json:"error_detail"`
	Classifications []Classification
}

// Classification holds the classification information related to a
// processed document
type Classification struct {
	TagName string
	TagID int  `json:"tag_id"`
	Confidence float64
}

func startTimer(name string) func() {
	t := time.Now()
	return func() {
		d := time.Now().Sub(t)
		log.Println(name, "took", d)
	}
}
