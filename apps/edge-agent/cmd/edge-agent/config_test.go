package main

import (
	"reflect"
	"testing"
)

func TestParseSubscribeTopics(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace", "   ", nil},
		{"single", "fast-gpu", []string{"fast-gpu"}},
		{"multi", "default, fast-gpu ,slow-gpu", []string{"default", "fast-gpu", "slow-gpu"}},
		{"trailing comma", "a,b,", []string{"a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSubscribeTopics(tc.raw)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseSubscribeTopics(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
