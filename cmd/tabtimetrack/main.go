package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
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
	typeDay   = 1
	typeWeek  = 2
	typeMonth = 3
	typeYear  = 4
	typeAll   = 5
)

func parseType(s string) (int32, error) {
	s = strings.ToLower(s)
	switch s {
	default:
		return 0, fmt.Errorf("unknown segment type %q, try: day|week|month|year|all", s)
	case "d", "day":
		return typeDay, nil
	case "w", "wk", "week":
		return typeWeek, nil
	case "m", "mo", "month":
		return typeMonth, nil
	case "y", "yr", "year":
		return typeYear, nil
	case "a", "all":
		return typeAll, nil
	}
}

func split(d civil.Date) []tabtimetrack.Code {
	t := d.In(time.Local)
	yr, wk := t.ISOWeek()
	return []tabtimetrack.Code{
		{Type: typeAll, Value: 0},
		{Type: typeDay, Value: int32(d.Year*10000 + int(d.Month)*100 + d.Day)},
		{Type: typeWeek, Value: int32(yr*100 + wk)},
		{Type: typeMonth, Value: int32(yr*100 + int(d.Month))},
		{Type: typeYear, Value: int32(yr)},
	}
}
func desc(code tabtimetrack.Code) string {
	v := code.Value
	switch code.Type {
	default:
		return ""
	case typeAll:
		return "Sum"
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
	rateString := flag.String("rate", "", "hourly rate")
	descTableLength := flag.Int("length", 50, "table description length, negative for unlimited")
	descTypeString := flag.String("desc", "", "show descriptions summarazed by day|week|month|year|all")
	outputTypeString := flag.String("ot", "tsv", "output type: tsv|csv")
	flag.Parse()
	// var rate *apd.Decimal
	var rate *big.Rat
	if len(*rateString) > 0 {
		var ok bool
		rate = big.NewRat(0, 100)
		rate, ok = rate.SetString(*rateString)
		if !ok {
			return fmt.Errorf("parse rate failed %q", *rateString)
		}
	}

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
	if limit == 0 {
		w.Line("Date", "hms", "dec", "Bill")
	} else {
		w.Line("Date", "hms", "dec", "Bill", "Description")
	}
	sums := tabtimetrack.SumFunc(f.List, split, desc)
	for _, sl := range sums {
		var amount string
		sl.CalcHours()
		hours := sl.Hours.FloatString(2)
		if rate != nil {
			sl.CalcAmount(rate)
			amount = sl.Amount.FloatString(2)
		}
		durs := sl.Duration.String()
		if limit == 0 {
			w.Line(sl.Name, durs, hours, amount)
		} else {
			descLine := strings.Join(tabtimetrack.SplitSortDeDuplicate(sl.Description, "."), " ")
			descLine = strings.ReplaceAll(descLine, "\t", " ")
			if limit > 0 && len(descLine) > limit {
				descLine = descLine[:limit] + "..."
			}
			w.Line(sl.Name, durs, hours, amount, descLine)
		}
	}

	if err := w.Close(); err != nil {
		return err
	}
	if len(*descTypeString) == 0 {
		return nil
	}
	descType, err := parseType(*descTypeString)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\nDescriptions:\n")
	printed := false

	for _, sl := range sums {
		if sl.Code.Type != descType {
			continue
		}
		descList := tabtimetrack.SplitSortDeDuplicate(sl.Description, ".")
		if len(descList) > 0 {
			printed = true
		}
		fmt.Fprintf(out, "\n%s:\n", desc(sl.Code))
		for i, d := range descList {
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
