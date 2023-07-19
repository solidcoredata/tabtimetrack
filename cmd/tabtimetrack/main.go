package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"cloud.google.com/go/civil"
	"github.com/kardianos/task"
	tabtimetrack "github.com/solidcoredata/tabtimetracker"
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

	w := tabwriter.NewWriter(os.Stdout, 2, 8, 2, '\t', 0)
	defer w.Flush()

	fmt.Fprintf(w, "Report:\t%s\n", f.Title)
	sums := tabtimetrack.SumFunc(f.List, split, desc)
	for _, sl := range sums {
		var amount string
		if rate != nil {
			product := tabtimetrack.MultiplyRate(rate, sl.Duration)
			amount = product.FloatString(2)
		}
		desc := strings.Join(sl.Description, " ")
		const limit = 50
		if len(desc) > limit {
			desc = desc[:limit] + "..."
		}
		fmt.Fprintf(w, "%s:\t%s\t%s\t%s\n", sl.Name, sl.Duration.String(), amount, desc)
	}

	return w.Flush()
}
