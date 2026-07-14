package flags

import (
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	type flagsStruct struct {
		Bool      bool     `flag:"--bool"`
		String    string   `flag:"--string"`
		Int       int      `flag:"--int"`
		Slice     []string `flag:"--slice"`
		SliceJoin []string `flag:"--slice-join|join:,"`
	}

	tests := []struct {
		name string
		v    any
		want []string
	}{
		{
			"bool",
			flagsStruct{Bool: true},
			[]string{"--bool"},
		},
		{
			"string",
			flagsStruct{String: "foo"},
			[]string{"--string", "foo"},
		},
		{
			"slice",
			flagsStruct{Slice: []string{"foo", "bar"}},
			[]string{"--slice", "foo", "--slice", "bar"},
		},
		{
			"slice join",
			flagsStruct{SliceJoin: []string{"foo", "bar"}},
			[]string{"--slice-join", "foo,bar"},
		},
		{
			"empty slice",
			flagsStruct{Slice: []string{}},
			[]string{},
		},
		{
			"empty slice join",
			flagsStruct{SliceJoin: []string{}},
			[]string{},
		},
		{
			"int",
			flagsStruct{Int: 10},
			[]string{"--int", "10"},
		},
		{
			"multiple",
			flagsStruct{Bool: true, String: "foo"},
			[]string{"--bool", "--string", "foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Encode(tt.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}
