package main

import (
	"reflect"
	"testing"
)

func TestSearchArgsReorder(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "flags after query are moved first",
			args:     []string{"invoice from microsoft", "-min-score", "0.5"},
			expected: []string{"-min-score", "0.5", "invoice from microsoft"},
		},
		{
			name:     "flags first returns unchanged",
			args:     []string{"-min-score", "0.5", "invoice from microsoft"},
			expected: []string{"-min-score", "0.5", "invoice from microsoft"},
		},
		{
			name:     "query only returns unchanged",
			args:     []string{"invoice from microsoft"},
			expected: []string{"invoice from microsoft"},
		},
		{
			name:     "empty args returns unchanged",
			args:     []string{},
			expected: []string{},
		},
		{
			name:     "multiple positionals then flags",
			args:     []string{"one", "two", "-limit", "5"},
			expected: []string{"-limit", "5", "one", "two"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchArgsReorder(tt.args)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("searchArgsReorder() = %v, want %v", got, tt.expected)
			}
		})
	}
}
