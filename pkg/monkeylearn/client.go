package monkeylearn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type DataObject struct {
	Text string `json:"text"`
	ExternalID *string `json:"external_id"`
}

type Documents struct {
	Data []DataObject `json:"data"`
}

func (d *Documents) Add(document string) *Documents {
	d.Data = append(d.Data, DataObject{Text: document})
	return d
}

type Client struct {
	token string
	client *http.Client
}

func NewClient(token string) *Client {
	return &Client{token: token, client: &http.Client{} }
}

// Rate limiting
// {
// 	"status_code": 429,
// 	"error_code": "CONCURRENCY_RATE_LIMIT",
// 	"detail": "Request was throttled. Too many concurrent requests."
// }

func (c *Client) Classify(model string, docs interface{}) *http.Response {
	defer startTimer(model)()

	url := "https://api.monkeylearn.com/v3"
	endpoint := fmt.Sprintf("%s/classifiers/%s/classify/", url, model)
	data, err := json.Marshal(docs)
	if err != nil { log.Panic(err) }

	log.Printf("%#v", string(data))

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil { log.Panic(err) }

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", c.token))
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil { log.Panic(err) }

	// X-Query-Limit-Limit 	Your current query limit
	// X-Query-Limit-Remaining Number of queries your account has left to use
	log.Printf(
		"Remaining API calls: %s / %s",
		resp.Header.Get("X-Query-Limit-Remaining"),
		resp.Header.Get("X-Query-Limit-Limit"),
	)
	return resp
}

func startTimer(name string) func() {
	t := time.Now()
	return func() {
		d := time.Now().Sub(t)
		log.Println(name, "took", d)
	}
}
