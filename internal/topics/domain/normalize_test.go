package domain

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
		{"nil becomes none", nil, []string{}},
		{"empty becomes none", []string{}, []string{}},
		{"empty strings are dropped", []string{"", " "}, []string{}},
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
