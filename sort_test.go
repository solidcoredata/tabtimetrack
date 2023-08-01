package tabtimetrack

import (
	"reflect"
	"testing"
)

func TestDeduplicate(t *testing.T) {
	list := []struct {
		Name   string
		Input  []string
		Output []string
	}{
		{"empty", []string{}, []string{}},
		{"noop", []string{"abc", "def"}, []string{"abc", "def"}},
		{"simple", []string{"abc", "abc", "def"}, []string{"abc", "def"}},
		{"sort", []string{"def", "abc", "abc"}, []string{"abc", "def"}},
		{"sort not empty", []string{"def", "abc", "abc", ""}, []string{"abc", "def"}},
		{"not empty", []string{""}, []string{}},
	}

	for _, item := range list {
		t.Run(item.Name, func(t *testing.T) {
			got := sortDeDuplicate(item.Input)
			if !reflect.DeepEqual(got, item.Output) {
				t.Fatalf("want %q, got %q", item.Output, got)
			}
		})
	}
}

func TestSplitStop(t *testing.T) {
	const stop = "."
	list := []struct {
		Name   string
		Input  []string
		Output []string
	}{
		{"empty", []string{}, []string{}},
		{"noop", []string{"abc.", "def."}, []string{"abc.", "def."}},
		{"simple", []string{"Hello World."}, []string{"Hello World."}},
		{"multiple", []string{"Hello World. Hello Block"}, []string{"Hello World.", "Hello Block."}},
		{"multiple stop", []string{"Hello World. Hello Block."}, []string{"Hello World.", "Hello Block."}},
		{"not empty", []string{""}, []string{}},
	}

	for _, item := range list {
		t.Run(item.Name, func(t *testing.T) {
			got := splitAppend(item.Input, stop)
			if !reflect.DeepEqual(got, item.Output) {
				t.Fatalf("want %q, got %q", item.Output, got)
			}
		})
	}
}
