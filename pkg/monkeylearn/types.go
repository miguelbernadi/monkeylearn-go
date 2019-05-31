package monkeylearn

import "fmt"

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
