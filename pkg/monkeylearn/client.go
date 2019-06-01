package monkeylearn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	hostname      = "https://api.monkeylearn.com"
	classifierURL = "/v3/classifiers/%s/classify/"
	extractorURL  = "/v3/extractors/%s/extract/"
)

type APILimits struct {
	RequestLimit, RequestRemaining int
	sync.Mutex
}

func (a *APILimits) updateLimits(limit, remains string) error {
	requests, err := strconv.Atoi(limit)
	if err != nil {
		return fmt.Errorf("error reading API query limit: %s", err)
	}

	remaining, err := strconv.Atoi(remains)
	if err != nil {
		return fmt.Errorf("error reading API query limit remaining: %s", err)
	}

	a.update(requests, remaining)
	return nil
}

func (a *APILimits) update(limit, remaining int) {
	a.Lock()
	a.RequestLimit = limit
	a.RequestRemaining = remaining
	a.Unlock()
}

// Client holds the authentication data to connect to the MonkeyLearn
// API and is used as gateway to operate with the API
type Client struct {
	client                  *http.Client
	token, server, endpoint string
	Limits                  *APILimits
	queue                   chan *http.Request
	results                 chan Result
	err                     chan error
	rate                    time.Duration
	docs                    []DataObject
}

// NewClient returns a new Client initialized with a custom HTTP
// client, and API token and a target hostname (e.g. proxying)
func NewClient(client *http.Client, token, hostname string) *Client {
	c := &Client{client: client, token: token, server: hostname}
	c.Limits = &APILimits{}
	c.queue = make(chan *http.Request)
	c.results = make(chan Result)
	c.err = make(chan error)
	return c
}

// NewDefaultClient returns a new Client initialized with an
// authentication token usable for the official API
func NewDefaultClient(token string) *Client {
	return NewClient(http.DefaultClient, token, hostname)
}

// SetMaxRate specifies how often to issue requests
func (c *Client) SetMaxRate(rate time.Duration) *Client {
	c.rate = rate
	return c
}

// SetClassificationModel configures which model to use for classification
func (c *Client) SetClassificationModel(model string) *Client {
	c.endpoint = fmt.Sprintf(classifierURL, model)
	return c

}

// SetExtractionModel configures which model to use for extraction
func (c *Client) SetExtractionModel(model string) *Client {
	c.endpoint = fmt.Sprintf(extractorURL, model)
	return c
}

// ProcessDocuments adds documents to be processed
func (c *Client) ProcessDocuments(docs ...DataObject) *Client {
	c.docs = append(c.docs, docs...)
	return c
}

// Batch specifies the batch size to use and splits the documents
// pending processing into batches
func (c *Client) Batch(size int) {
	batches := SplitInBatches(c.docs, size)
	for _, b := range batches {
		data, _ := json.Marshal(b)
		c.queueRequest(c.endpoint, data)
	}
	close(c.queue)
}

// Results returns a channel where we can receive all the results from
// API calls
func (c *Client) Results() (<-chan Result, <-chan error) {
	return c.results, c.err
}

// Process does the appropriate call to the MonkeyLearn API and
// handles the response
func (c *Client) RunProcessor() {
	throttle := time.Tick(c.rate)
	for req := range c.queue {
		<-throttle // rate limit
		go func(req *http.Request) {
			resp, err := c.executeRequest(req)
			if err != nil {
				c.err <- err
			} else {
				// Only if request is successful
				if err := c.Limits.updateLimits(
					resp.Header.Get("X-Query-Limit-Limit"),
					resp.Header.Get("X-Query-Limit-Remaining"),
				); err != nil {
					c.err <- fmt.Errorf(
						"error reading request limits: %s",
						err,
					)
				}
				res, err := deserializeResponse(resp)
				if err != nil {
					c.err <- fmt.Errorf(
						"error deserializing API response: %s",
						err,
					)
				}

				for _, result := range res {
					c.results <- result
				}
			}
		}(req)
	}
	// No one else can write in the channel after us
	close(c.results)
	close(c.err)
}

// queueRequest adds a Request for later processing
func (c *Client) queueRequest(endpoint string, data []byte) {
	req := c.newRequest(c.server+endpoint, data)
	c.queue <- req
}

func (c *Client) executeRequest(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// We get rate limited. Do something
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("request got ratelimited calling %s", req.URL)
	}

	// Not succesful? Better error out
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unsuccessful request: %s got %s", req.URL, resp.Status)
	}

	return resp, nil
}

func (c *Client) newRequest(url string, data []byte) *http.Request {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", c.token))
	req.Header.Add("Content-Type", "application/json")

	return req
}
