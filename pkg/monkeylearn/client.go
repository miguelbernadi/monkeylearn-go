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

const (
	hostname      = "https://api.monkeylearn.com"
	classifierURL = "/v3/classifiers/%s/classify/"
	extractorURL  = "/v3/extractors/%s/extract/"
)

// Client holds the authentication data to connect to the MonkeyLearn
// API and is used as gateway to operate with the API
type Client struct {
	client                         *http.Client
	token, server, endpoint        string
	RequestLimit, RequestRemaining int
	queue                          chan *http.Request
	results                        chan Result
	rate                           time.Duration
	docs                           []DataObject
}

// NewClient returns a new Client initialized with a custom HTTP
// client, and API token and a target hostname (e.g. proxying)
func NewClient(client *http.Client, token, hostname string) *Client {
	c := &Client{client: client, token: token, server: hostname}
	c.queue = make(chan *http.Request)
	c.results = make(chan Result)
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

func (c *Client) SetClassificationModel(model string) *Client {
	c.endpoint = fmt.Sprintf(classifierURL, model)
	return c

}
func (c *Client) SetExtrationModel(model string) *Client {
	c.endpoint = fmt.Sprintf(extractorURL, model)
	return c
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

// Process does the appropriate call to the MonkeyLearn API and
// handles the response
func (c *Client) RunProcessor() {
	throttle := time.Tick(c.rate)
	for req := range c.queue {
		<-throttle // rate limit
		go func(req *http.Request) {
			resp, err := c.executeRequest(req)
			if err != nil {
				log.Println(err)
			} else {
				// Only if request is successful
				if err := c.updateLimits(resp); err != nil {
					log.Println(fmt.Errorf("error reading request limits: %s", err))
				}
				res, err := deserializeResponse(resp)
				if err != nil {
					log.Println(fmt.Errorf("error deserializing API response: %s", err))
				}

				for _, result := range res {
					c.results <- result
				}
			}
		}(req)
	}
	// No one else can write in the channel after us
	close(c.results)
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

func (c *Client) updateLimits(response *http.Response) error {
	var err error
	c.RequestRemaining, err = strconv.Atoi(response.Header.Get("X-Query-Limit-Remaining"))
	if err != nil {
		return fmt.Errorf("error reading API query limit remaining: %s", err)
	}
	c.RequestLimit, err = strconv.Atoi(response.Header.Get("X-Query-Limit-Limit"))
	if err != nil {
		return fmt.Errorf("error reading API query limit: %s", err)
	}
	return nil
}

func (c *Client) ProcessDocuments(docs ...DataObject) *Client {
	c.docs = append(c.docs, docs...)
	return c
}

func (c *Client) Batch(size int) {
	batches := SplitInBatches(c.docs, size)
	for _, b := range batches {
		data, _ := json.Marshal(b)
		c.queueRequest(c.endpoint, data)
	}
	close(c.queue)
}

func (c *Client) Results() <-chan Result {
	return c.results
}

// Result holds the results of processing a document be it either an
// extraction or a classification
type Result struct {
	Text            string
	ExternalID      string `json:"external_id"`
	IsError         bool   `json:"error"`
	ErrorDetail     string `json:"error_detail"`
	Classifications []Classification
	Extractions     []Extraction
}

// Error returns an error if the processing had an error
func (r Result) Error() error {
	if r.IsError {
		return fmt.Errorf(r.ErrorDetail)
	}
	return nil
}

// MergeResultList returns a slice of Result resulting of merging a
// series of Result slices
func MergeResultList(lists ...[]Result) []Result {
	dict := make(map[string]Result)

	for _, list := range lists {
		for _, result := range list {
			index := result.ExternalID
			val, ok := dict[index]
			if !ok {
				dict[index] = result
			} else {
				dict[index] = mergeResult(val, result)
			}
		}
	}

	values := make([]Result, 0, len(dict))
	for _, v := range dict {
		values = append(values, v)
	}
	return values
}

func mergeResult(a, b Result) Result {
	a.Classifications = append(a.Classifications, b.Classifications...)
	a.Extractions = append(a.Extractions, b.Extractions...)
	return a
}

// Classification holds the classification information related to a
// processed document
type Classification struct {
	TagName    string `json:"tag_name"`
	TagID      int    `json:"tag_id"`
	Confidence float64
}

// Extraction represents an instance of extracted elements from a
// document
type Extraction struct {
	TagName       string      `json:"tag_name"`
	ExtractedText string      `json:"extracted_text"`
	OffsetSpan    []int       `json:"offset_span"`
	ParsedValue   interface{} `json:"parsed_value"`
}

func startTimer(name string) func() {
	t := time.Now()
	return func() {
		d := time.Since(t)
		log.Println(name, "took", d)
	}
}

func deserializeResponse(response *http.Response) ([]Result, error) {
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var res []Result
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
