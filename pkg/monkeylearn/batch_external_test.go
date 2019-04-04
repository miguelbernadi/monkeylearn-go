package monkeylearn_test

import (
	"testing"

	"github.com/miguelbernadi/monkeylearn-go/pkg/monkeylearn"
)

func TestSplitInBatches(t *testing.T) {
	docs :=  []monkeylearn.DataObject{
		{
			Text: "obladi",
		},
		{
			Text: "oblada",
		},
	}
	var table = []struct{
		batchsize int
		docs []monkeylearn.DataObject
		result []monkeylearn.Batch
	}{
		{
			batchsize: 1,
			docs: docs,
			result: []monkeylearn.Batch{
				{
					Data: []monkeylearn.DataObject{
						{
							Text: "obladi",
						},
					},
				},
				{
					Data: []monkeylearn.DataObject{
						{
							Text: "oblada",
						},
					},
				},
			},
		},
		{
			batchsize: 2,
			docs: docs,
			result: []monkeylearn.Batch{
				{
					Data: []monkeylearn.DataObject{
						{
							Text: "obladi",
						},
						{
							Text: "oblada",
						},
					},
				},
			},
		},
	}

	for _, tc := range table {
		result := monkeylearn.SplitInBatches(tc.docs, tc.batchsize)
		if len(result) != len(tc.result) {
			t.Errorf(
				"Mismatched number of batches,"+
					" expected %d but got %d",
				len(tc.result),
				len(result),
			)
		}
		for i, r := range result {
			curData := tc.result[i].Data
			
			if len(r.Data) != len(curData) {
				t.Errorf("Mismatched batch sizes,"+
					"expected %d but got %d",
					len(curData),
					len(r.Data),
				)
			}
			
			for j, d := range r.Data {
				if d != curData[j] {
					t.Errorf("Mismatched batch contents")
				}
			}
		}
	}
}
