package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/kardianos/task"
	tabtimetrack "github.com/solidcoredata/tabtimetracker"
	"github.com/solidcoredata/tabtimetracker/datatable"
)

func main() {
	err := task.Start(context.Background(), time.Second*3, run)
	if err != nil {
		log.Fatal(err)
	}
}

const (
	typeDay           = 1
	typeWeek          = 2
	typeMonth         = 3
	typeYear          = 4
	typeAll           = 5
	typeAllNoBreakout = 6
	typeBreakout      = 11
)

const segmentType = "day(d)|week(wk)|month(mo)|year(yr)|all(a)|all-breakout(ab)|breakout(br)"

func parseType(list string) ([]int32, error) {
	ss := strings.Split(list, ",")
	var tt []int32
	for _, s := range ss {
		s = strings.ToLower(s)
		switch s {
		default:
			return tt, fmt.Errorf("unknown segment type %q, try: %s", s, segmentType)
		case "d", "day":
			tt = append(tt, typeDay)
		case "w", "wk", "week":
			tt = append(tt, typeWeek)
		case "m", "mo", "month":
			tt = append(tt, typeMonth)
		case "y", "yr", "year":
			tt = append(tt, typeYear)
		case "a", "all":
			tt = append(tt, typeAll)
		case "ab", "all-breakout":
			tt = append(tt, typeAllNoBreakout, typeBreakout)
		case "br", "breakout":
			tt = append(tt, typeBreakout)
		}
	}
	return tt, nil
}

func NewCoder(breakout []tabtimetrack.Task) *coder {
	c := &coder{
		breakoutReferenceLookup:  make(map[string]Task),
		breakoutAssignmentLookup: make(map[int32]Task),
	}
	var nv int32
	for _, b := range breakout {
		nv++

		t := Task{
			Value: nv,
			Task:  b,
		}
		c.breakoutReferenceLookup[b.Reference] = t
		c.breakoutAssignmentLookup[nv] = t
	}
	return c
}

type Task struct {
	Value int32
	tabtimetrack.Task
}
type coder struct {
	breakoutReferenceLookup  map[string]Task
	breakoutAssignmentLookup map[int32]Task
}

var _ tabtimetrack.Coder = &coder{}

func (c *coder) Split(d civil.Date, taskList []tabtimetrack.Task) ([]tabtimetrack.Code, error) {
	var differentReference bool
	var breakoutValue int32
	for _, t := range taskList {
		b := c.breakoutReferenceLookup[t.Reference].Value
		if b == 0 {
			continue
		}
		if breakoutValue == 0 {
			breakoutValue = b
		}
		if breakoutValue != b {
			differentReference = true
		}
	}
	if breakoutValue > 0 {
		var err error
		if differentReference {
			err = fmt.Errorf("multiple different breakout references on same time line cannot be split")
		}
		return []tabtimetrack.Code{
			{Type: typeAll, Value: 0},
			{Type: typeBreakout, Value: breakoutValue},
		}, err
	}
	t := d.In(time.Local)
	yr, wk := t.ISOWeek()
	return []tabtimetrack.Code{
		{Type: typeAll, Value: 0},
		{Type: typeAllNoBreakout, Value: 0},
		{Type: typeDay, Value: int32(d.Year*10000 + int(d.Month)*100 + d.Day)},
		{Type: typeWeek, Value: int32(yr*100 + wk)},
		{Type: typeMonth, Value: int32(yr*100 + int(d.Month))},
		{Type: typeYear, Value: int32(yr)},
	}, nil
}
func (c *coder) Describe(code tabtimetrack.Code) string {
	v := code.Value
	switch code.Type {
	default:
		return ""
	case typeBreakout:
		t, ok := c.breakoutAssignmentLookup[v]
		if !ok {
			return fmt.Sprintf("unknown breakout code %d", v)
		}
		if len(t.Description) > 0 {
			return t.Description
		}
		return t.Reference
	case typeAll:
		return "Sum"
	case typeAllNoBreakout:
		return "Sum-Breakout"
	case typeDay:
		yr := v / 10000
		md := v - yr*10000
		mo := md / 100
		dy := md - mo*100
		return fmt.Sprintf("%04d-%02d-%02d", yr, mo, dy)
	case typeWeek:
		yr := v / 100
		wk := v - yr*100
		return fmt.Sprintf("%04d-wk%02d", yr, wk)
	case typeMonth:
		yr := v / 100
		mo := v - yr*100
		return fmt.Sprintf("%04d-mo%02d", yr, mo)
	case typeYear:
		return fmt.Sprintf("%04d", v)
	}
}

func run(ctx context.Context) error {
	fn := flag.String("f", "", "filename to open")
	descTableLength := flag.Int("length", 50, "table description length, negative for unlimited")
	descTypeString := flag.String("desc", "", "show descriptions summarazed by "+segmentType)
	outputTypeString := flag.String("ot", "tsv", "output type: tsv|csv")
	flag.Parse()

	bb, err := os.ReadFile(*fn)
	if err != nil {
		return err
	}
	f, err := tabtimetrack.Parse(bb)
	if err != nil {
		return err
	}

	var out = os.Stdout

	var w datatable.Writer
	switch *outputTypeString {
	default:
		return fmt.Errorf("unknown output type (ot): %q", *outputTypeString)
	case "tsv":
		w = datatable.NewTSV(out)
	case "csv":
		w = datatable.NewCSV(out)
	}
	defer w.Close()

	limit := *descTableLength

	w.Line("Report:", f.Title)
	w.Line("Rate:", f.Rate.FloatString(2))
	w.Line()
	if limit == 0 {
		w.Line("Date", "hms", "dec", "Bill")
	} else {
		w.Line("Date", "hms", "dec", "Bill", "Description")
	}

	c := NewCoder(f.Breakout)
	sums, sumError := tabtimetrack.SumFunc(f.List, c)
	for _, sl := range sums {
		var amount string
		sl.CalcHours()
		hours := sl.Hours.FloatString(2)
		if f.Rate != nil {
			sl.CalcAmount(f.Rate)
			amount = sl.Amount.FloatString(2)
		}
		durs := sl.Duration.String()
		if limit == 0 {
			w.Line(sl.Name, durs, hours, amount)
		} else {
			refLine := strings.Join(sl.Reference, ",")
			descLine := strings.Join(sl.Description, " ")
			descLine = strings.ReplaceAll(descLine, "\t", " ")
			if len(refLine) > 0 {
				descLine = "[" + refLine + "] " + descLine
			}
			if limit > 0 && len(descLine) > limit {
				descLine = descLine[:limit] + "..."
			}
			w.Line(sl.Name, durs, hours, amount, descLine)
		}
	}

	if err := w.Close(); err != nil {
		return err
	}

	if sumError != nil {
		fmt.Fprintf(out, "\nErrors:\n%v\n", sumError)
	}

	if len(*descTypeString) == 0 {
		return nil
	}
	descTypeList, err := parseType(*descTypeString)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\nDescriptions:\n")
	printed := false
	descTypeLookup := make(map[int32]bool, len(descTypeList))
	for _, item := range descTypeList {
		descTypeLookup[item] = true
	}

	for _, sl := range sums {
		if !descTypeLookup[sl.Code.Type] {
			continue
		}
		if len(sl.Description) > 0 || len(sl.Reference) > 0 {
			printed = true
		}
		fmt.Fprintf(out, "\n%s: (hr: %s, total: %s)\n", c.Describe(sl.Code), sl.Hours.FloatString(2), sl.Amount.FloatString(2))
		if len(sl.Reference) > 0 {
			fmt.Print("[")
		}
		for i, r := range sl.Reference {
			if i > 0 {
				fmt.Fprint(out, ",")
			}
			fmt.Fprint(out, r)
		}
		if len(sl.Reference) > 0 {
			fmt.Print("]")
			if len(sl.Reference) > 0 {
				fmt.Print(" ")
			}
		}
		for i, d := range sl.Description {
			if i > 0 {
				fmt.Fprint(out, " ")
			}
			fmt.Fprint(out, d)
		}
		fmt.Fprint(out, "\n")
	}
	if !printed {
		fmt.Fprintf(out, "No descriptions")
	}

	return nil
}
