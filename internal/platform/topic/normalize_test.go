package topic

import (
	"reflect"
	"testing"
)

func TestNormalizeTopics(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil becomes default", nil, []string{DefaultKey}},
		{"empty becomes default", []string{}, []string{DefaultKey}},
		{"empty strings become default", []string{"", " "}, []string{DefaultKey}},
		{"dedupe preserves order", []string{"a", "b", "a"}, []string{"a", "b"}},
		{"keeps default with others", []string{DefaultKey, "x"}, []string{DefaultKey, "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeTopics(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("NormalizeTopics(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
