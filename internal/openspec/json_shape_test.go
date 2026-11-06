package openspec_test

import (
	"strings"
	"testing"
)

func TestShowChangeMalformedDelta(t *testing.T) {
	req := `"requirements":[{"name":"R","scenarios":[]}]`
	for name, d := range map[string]string{
		"unknown operation":   `{"spec":"booking","operation":"MOVED",` + req + `}`,
		"empty spec":          `{"spec":"","operation":"ADDED",` + req + `}`,
		"renamed no rename":   `{"spec":"booking","operation":"RENAMED"}`,
		"renamed empty from":  `{"spec":"booking","operation":"RENAMED","rename":{"from":"","to":"B"}}`,
		"renamed empty to":    `{"spec":"booking","operation":"RENAMED","rename":{"from":"A","to":""}}`,
		"added no req":        `{"spec":"booking","operation":"ADDED"}`,
		"modified empty name": `{"spec":"booking","operation":"MODIFIED","requirements":[{"name":""}]}`,
		"removed no req":      `{"spec":"booking","operation":"REMOVED","requirements":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := jsonRunner(t, 0, `{"deltas":[`+d+`]}`, changeArgs)
			if _, err := r.ShowChange("add-x"); err == nil || !strings.Contains(err.Error(), "openspec show add-x") {
				t.Fatalf("err = %v", err)
			}
		})
	}
	ok := `{"deltas":[{"spec":"b","operation":"RENAMED","rename":{"from":"A","to":"B"}},{"spec":"b","operation":"REMOVED",` + req + `}]}`
	if got, err := jsonRunner(t, 0, ok, changeArgs).ShowChange("add-x"); err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
}
