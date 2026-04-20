package wares_test

import (
	"testing"

	"github.com/dimmerz92/wares"
)

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{name: "no values"},
		{name: "empty slice", values: []string{}},
		{name: "one blank string", values: []string{""}},
		{name: "one string", values: []string{"1"}, expected: "1"},
		{name: "first of three", values: []string{"1", "2", "3"}, expected: "1"},
		{name: "second of three", values: []string{"", "2", "3"}, expected: "2"},
		{name: "third of three", values: []string{"", "", "3"}, expected: "3"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := wares.Coalesce(test.values...)
			if got != test.expected {
				t.Errorf("expected %s, got %s", test.expected, got)
			}
		})
	}
}

func TestIIF(t *testing.T) {
	tests := []struct {
		name      string
		condition bool
		v1        string
		v2        string
		expected  string
	}{
		{name: "false condition", v1: "true", v2: "false", expected: "false"},
		{name: "true condition", condition: true, v1: "true", v2: "false", expected: "true"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := wares.IIF(test.condition, test.v1, test.v2)
			if got != test.expected {
				t.Errorf("expected %s, got %s", test.expected, got)
			}
		})
	}
}
