package utils

import (
	"encoding/json"
	"testing"
)

func TestMergeMaps(t *testing.T) {
	type testCase struct {
		src    string
		dest   string
		merged string
	}
	testCases := []testCase{{
		dest: `{
			"a":1,
			"b":{
				"c":{
					"d":"e"
				},
				"f":"g"
			}
		}`,
		src: `{
			"b":{
				"c": 2
			}
		}`,
		merged: `{
			"a":1,
			"b":{
				"c":2,
				"f":"g"
			}
		}`,
	}}

	for _, t0 := range testCases {
		srcMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(t0.src), &srcMap)
		if err != nil {
			t.Fatal(err)
		}
		destMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(t0.dest), &destMap)
		if err != nil {
			t.Fatal(err)
		}
		out := make(map[string]interface{})
		err = json.Unmarshal([]byte(t0.merged), &out)
		if err != nil {
			t.Fatal(err)
		}
		outB, err := json.MarshalIndent(out, " ", "")
		if err != nil {
			t.Fatal(err)
		}
		MergeMaps(srcMap, destMap)
		merged, err := json.MarshalIndent(destMap, " ", "")
		if err != nil {
			t.Fatal(err)
		}
		if string(outB) != string(merged) {
			t.Errorf("Expected %s but got %s", string(outB), string(merged))
		}
	}
}
