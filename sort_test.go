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

func TestSplitTask(t *testing.T) {
	const stop = "."
	list := []struct {
		Name   string
		Input  string
		Output []Task
	}{
		{"empty", "", []Task{}},
		{"simple", "abc. def.", []Task{{Description: "abc."}, {Description: "def."}}},
		{"code", "[123] abc. [456] def.", []Task{{Reference: "123", Description: "abc."}, {Reference: "456", Description: "def."}}},
	}

	for _, item := range list {
		t.Run(item.Name, func(t *testing.T) {
			got := splitDescription(item.Input, stop, ensureStop)
			if !reflect.DeepEqual(got, item.Output) {
				t.Fatalf("want %q, got %q", item.Output, got)
			}
		})
	}
}
