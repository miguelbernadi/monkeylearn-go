package monkeylearn_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/miguelbernadi/monkeylearn-go/pkg/monkeylearn"
)

// Holds a map of strings to results for testing
type serverResponses map[string]monkeylearn.Result

// Creates a server that takes a serverResponses map to provide answers
func testServer(responseList serverResponses) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					// Cannot read request body
					http.Error(
						w,
						"",
						http.StatusInternalServerError,
					)
				}

				// Deserialize request
				batch := monkeylearn.NewBatch()
				err = json.Unmarshal(body, &batch)
				if err != nil {
					http.Error(w, "", http.StatusBadRequest)
				}

				// Verify request is non-empty
				if len(batch.Data) == 0 {
					// empty requests are refused
					http.Error(w, "", http.StatusBadRequest)
				}

				// This should build a response per
				// document if there are more
				// documents in the Batch
				response, ok := responseList[batch.Data[0].Text]
				if !ok {
					http.Error(
						w,
						"",
						http.StatusInternalServerError,
					)
				}
				result := []monkeylearn.Result{response}

				// Respond to client
				data, err := json.Marshal(result)
				if err != nil {
					http.Error(
						w,
						"",
						http.StatusInternalServerError,
					)
				}
				io.Copy(w, bytes.NewReader(data))
			},
		),
	)
}

// Global map of result cases
var respList = serverResponses{
	"obladi": {
		Classifications: []monkeylearn.Classification{},
	},
	"oblada": {
		Extractions: []monkeylearn.Extraction{},
	},
}

func TestBatchClassify(t *testing.T) {
	// Initialize test server
	ts := testServer(respList)
	defer ts.Close()

	// Initialize library client
	client := monkeylearn.NewClient(ts.Client(), "test-token", ts.URL)

	// Create a Batch of documents to request
	batch := monkeylearn.Batch{}
	batch.Add(monkeylearn.DataObject{ Text: "obladi" })

	// Request Classification
	resp, err := batch.Classify("classifier", client)
	if err != nil {
		// Server is not expected to fail
		t.Fatalf("Unexpected error: %s", err)
	}

	// Verify returned data
	if len(resp) != len(batch.Data) {
		t.Errorf(
			"Mismatched batch size and results,"+
				"expected %d but got %d",
			len(batch.Data),
			len(resp),
		)
	}
	for _, r := range resp {
		if r.Error() != nil {
			t.Errorf("unexpected error from API: %s", r.Error())
		}
	}
}

func TestBatchExtract(t *testing.T) {
	// Initialize test server
	ts := testServer(respList)
	defer ts.Close()

	// Initialize library client
	client := monkeylearn.NewClient(ts.Client(), "test-token", ts.URL)

	// Create a Batch of documents to request
	batch := monkeylearn.Batch{}
	batch.Add(monkeylearn.DataObject{ Text: "oblada" })

	// Request Extraction
	resp, err := batch.Extract("extractor", client)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Verify returned data
	if len(resp) != len(batch.Data) {
		t.Errorf(
			"Mismatched batch size and results,"+
				"expected %d but got %d",
			len(batch.Data),
			len(resp),
		)
	}
	for _, r := range resp {
		if r.Error() != nil {
			t.Errorf("unexpected error from API: %s", r.Error())
		}
	}
}
