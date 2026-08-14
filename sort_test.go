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
		{"cap off", "[c] abc.", []Task{{Reference: "c", Description: "abc."}}},
	}

	for _, item := range list {
		t.Run(item.Name, func(t *testing.T) {
			got := splitDescription(item.Input, stop, ensureStop, "")
			if !reflect.DeepEqual(got, item.Output) {
				t.Fatalf("want %+v, got %+v", item.Output, got)
			}
		})
	}
}

func TestSplitTaskCapitalized(t *testing.T) {
	const stop = "."
	list := []struct {
		Name   string
		Input  string
		Output []Task
	}{
		{"cap only", "[c] abc.", []Task{{Description: "abc.", Capitalized: true}}},
		{"cap upper", "[C] abc.", []Task{{Description: "abc.", Capitalized: true}}},
		{"cap then code", "[c] [123] abc.", []Task{{Reference: "123", Description: "abc.", Capitalized: true}}},
		{"code then cap", "[123] [c] abc.", []Task{{Reference: "123", Description: "abc.", Capitalized: true}}},
		{"mixed", "[c] abc. def.", []Task{{Description: "abc.", Capitalized: true}, {Description: "def."}}},
		{"no cap", "[123] abc.", []Task{{Reference: "123", Description: "abc."}}},
	}

	for _, item := range list {
		t.Run(item.Name, func(t *testing.T) {
			got := splitDescription(item.Input, stop, ensureStop, "c")
			if !reflect.DeepEqual(got, item.Output) {
				t.Fatalf("want %+v, got %+v", item.Output, got)
			}
		})
	}
}
