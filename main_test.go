package main

import (
	"reflect"
	"testing"
)

func TestGroup(t *testing.T) {
	tests := []struct {
		name   string
		elems  []string
		groups map[string]map[string][]string
	}{
		{
			name:   "Simple",
			elems:  []string{"a_b_c", "a_b_d"},
			groups: map[string]map[string][]string{"a": map[string][]string{"b": []string{"a_b_c", "a_b_d"}}},
		},
		{
			name:  "Complex",
			elems: []string{"a_b_c", "a_b_d", "b_c_d", "b_d_e", "c"},
			groups: map[string]map[string][]string{
				"a": map[string][]string{
					"b": []string{
						"a_b_c", "a_b_d"}},
				"b": map[string][]string{
					"c": []string{
						"b_c_d"},
					"d": []string{
						"b_d_e"},
				},
				"c": map[string][]string{
					"c": []string{"c"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := group(test.elems)
			if !reflect.DeepEqual(result, test.groups) {
				t.Errorf("Test %q failed:\nExpected: %+v\nActual: %+v", test.name, test.groups, result)
			}
		})

	}
}
