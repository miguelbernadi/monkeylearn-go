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
	classifierURL = "https://api.monkeylearn.com/v3/classifiers/%s/classify/"
	extractorURL = "https://api.monkeylearn.com/v3/extractors/%s/extract/"
)

// Client holds the authentication data to connect to the MonkeyLearn
// API and is used as gateway to operate with the API
type Client struct {
	http.Client
	token string
	RequestLimit, RequestRemaining int
}

// NewClient returns a new Client initialized with an authentication token
func NewClient(token string) *Client {
	return &Client{token: token}
}

// Process does the appropriate call to the MonkeyLearn API and
// handles the response
func (c *Client) Process(model, fmtURL string, data []byte) ([]Result, error) {
	resp, err := c.Do(
		c.newRequest(
			fmt.Sprintf(fmtURL, model),
			data,
		),
	)
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
	c.updateLimits(resp)

	res, err := deserializeResponse(resp)
	if err != nil { log.Panic(err) }

	return res, nil
}

func (c *Client) newRequest(url string, data []byte) *http.Request {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil { log.Panic(err) }

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", c.token))
	req.Header.Add("Content-Type", "application/json")

	return req
}

func (c *Client) updateLimits(response *http.Response) {
	var err error
	c.RequestRemaining, err = strconv.Atoi(response.Header.Get("X-Query-Limit-Remaining"))
	if err != nil { log.Panic(err) }
	c. RequestLimit, err = strconv.Atoi(response.Header.Get("X-Query-Limit-Limit"))
	if err != nil { log.Panic(err) }
}

// Result holds the results of processing a document be it either an
// extraction or a classification
type Result struct {
	Text string
	ExternalID int `json:"external_id"`
	IsError bool `json:"error"`
	ErrorDetail string `json:"error_detail"`
	Classifications []Classification
	Extractions []Extraction
}

// Error returns an error if the processing had an error
func (r Result) Error() error {
	if r.IsError {
		return fmt.Errorf(r.ErrorDetail)
	}
	return nil
}

func MargeResultList(lists ...[]Result) []Result {
	dict := make(map[int]Result)

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
	TagName string `json:"tag_name"`
	TagID int  `json:"tag_id"`
	Confidence float64
}

// Extraction represents an instance of extracted elements from a
// document
type Extraction struct {
	TagName string `json:"tag_name"`
	ExtractedText int `json:"extracted_text"`
	OffsetSpan []int `json:"offset_span"`
	ParsedValue interface{} `json:"parsed_value"`
}

func startTimer(name string) func() {
	t := time.Now()
	return func() {
		d := time.Now().Sub(t)
		log.Println(name, "took", d)
	}
}

func deserializeResponse(response *http.Response) ([]Result, error) {
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil { return nil, err }

	var res []Result
	err = json.Unmarshal(body, &res)
	if err != nil { return nil, err }

	return res, nil
}
