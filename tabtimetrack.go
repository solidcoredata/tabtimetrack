package tabtimetrack

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/wadey/go-rounding"
)

type File struct {
	Title    string
	List     []Line
	Breakout []Task
	Rate     *big.Rat
}

type Task struct {
	Reference   string
	Description string
}

type Line struct {
	Number      int
	Date        civil.Date
	Start       civil.Time // Inclusive.
	Stop        civil.Time // Inclusive.
	Description string
	Duration    time.Duration
	TaskList    []Task
}

func timeDur(t civil.Time) time.Duration {
	return time.Duration(t.Hour)*time.Hour + time.Duration(t.Minute)*time.Minute + time.Duration(t.Second)*time.Second + time.Duration(t.Nanosecond)*time.Nanosecond
}

func SubTime(end, start civil.Time) time.Duration {
	ed := timeDur(end)
	sd := timeDur(start)
	return ed - sd
}

func ParseTime(bb []byte) (civil.Time, error) {
	var ct civil.Time
	if len(bb) == 0 {
		return ct, fmt.Errorf("empty time")
	}
	pp := bytes.Split(bb, []byte{':'})
	for i, p := range pp {
		n, err := strconv.ParseInt(string(p), 10, 32)
		v := int(n)
		if err != nil {
			return ct, fmt.Errorf("time part %d in %q: %w", i+1, bb, err)
		}
		switch i {
		case 0:
			ct.Hour = v
		case 1:
			ct.Minute = v
		case 2:
			ct.Second = v
		}
	}
	return ct, nil
}

func Parse(data []byte) (f File, err error) {
	const stop = "."
	ll := bytes.Split(data, []byte{'\n'})
	for i, l := range ll {
		ln := i + 1

		ww := bytes.Split(l, []byte{'\t'})
		if i == 0 && len(ww) == 1 {
			f.Title = string(ww[0])
			continue
		}
		if len(ww) == 1 && len(ww[0]) == 0 {
			continue
		}
		if ww0 := ww[0]; len(ww0) > 0 {
			switch ww0[0] {
			case '@':
				switch string(ww[0]) {
				default:
					return f, fmt.Errorf("line %d: unknown command %s", ln, ww0)
				case "@rem":
					continue
				case "@breakout":
					if len(ww) != 2 {
						return f, fmt.Errorf("line %d: missing expected breakout description", ln)
					}
					bb := splitDescription(string(ww[1]), stop, ensureNoStop)
					if len(bb) == 0 {
						return f, fmt.Errorf("line %d: missing expected breakout description", ln)
					}
					f.Breakout = append(f.Breakout, bb...)
				case "@rate":
					if f.Rate != nil {
						return f, fmt.Errorf("line %d: multiple rates per file not allowed", ln)
					}
					if len(ww) != 2 {
						return f, fmt.Errorf("line %d: missing rate value", ln)
					}
					var ok bool
					rateString := string(ww[1])
					rate := big.NewRat(0, 100)
					rate, ok = rate.SetString(rateString)
					if !ok {
						return f, fmt.Errorf("line %d: parse rate failed %q", ln, rateString)
					}
					f.Rate = rate
				}
				continue
			}
		}

		if len(ww) < 3 {
			return f, fmt.Errorf("line %d: incomplete line", ln)
		}
		d, err := civil.ParseDate(string(ww[0]))
		if err != nil {
			return f, fmt.Errorf("line %d: invalid date %w", ln, err)
		}
		ts, err := ParseTime(ww[1])
		if err != nil {
			return f, fmt.Errorf("line %d: start time %w", ln, err)
		}
		te, err := ParseTime(ww[2])
		if err != nil {
			return f, fmt.Errorf("line %d: end time %w", ln, err)
		}
		var desc string
		if len(ww) > 3 {
			desc = string(ww[3])
		}
		dur := SubTime(te, ts)
		if dur < 0 {
			return f, fmt.Errorf("line %d: duration negative, end time before start time", ln)
		}
		const maxLine = 10 * time.Hour
		if dur > maxLine {
			return f, fmt.Errorf("line %d: duration larger then %s, this must be a mistake", ln, maxLine)
		}
		f.List = append(f.List, Line{
			Number:      ln,
			Date:        d,
			Start:       ts,
			Stop:        te,
			Duration:    dur,
			Description: desc,
			TaskList:    splitDescription(desc, stop, ensureStop),
		})
	}
	sort.Slice(f.List, func(i, j int) bool {
		a, b := f.List[i], f.List[j]
		if a.Date != b.Date {
			return a.Date.Before(b.Date)
		}
		if a.Start != b.Start {
			return a.Start.Before(b.Start)
		}
		return a.Stop.Before(b.Stop)
	})
	var prev Line
	var errList error
	for i, item := range f.List {
		if i == 0 {
			prev = item
			continue
		}
		if item.Date != prev.Date {
			prev = item
			continue
		}
		if prev.Stop == item.Start || prev.Stop.After(item.Start) {
			errList = errors.Join(errList, fmt.Errorf("line %d overlaps line %d, ensure start and stop are not the same", prev.Number, item.Number))
		}
		prev = item
	}
	return f, errList
}

type Code struct {
	Type  int32
	Value int32
}

type SumLine struct {
	Code        Code
	Name        string
	Duration    time.Duration
	Hours       *big.Rat
	Amount      *big.Rat
	Reference   []string
	Description []string
}

func (sl *SumLine) CalcHours() {
	sl.Hours = big.NewRat(int64(sl.Duration), int64(time.Hour))
	rounding.Round(sl.Hours, 2, rounding.HalfEven)
}
func (sl *SumLine) CalcAmount(rate *big.Rat) {
	if sl.Hours == nil {
		sl.CalcHours()
	}
	sl.Amount = big.NewRat(0, 100)
	sl.Amount.Mul(rate, sl.Hours)
}

type Coder interface {
	Split(d civil.Date, tasks []Task) ([]Code, error)
	Describe(code Code) string
}

func SumFunc(lineList []Line, coder Coder) ([]*SumLine, error) {
	sums := make(map[Code]*SumLine, 100)
	var err error
	for _, line := range lineList {
		cc, errLine := coder.Split(line.Date, line.TaskList)
		if errLine != nil {
			err = errors.Join(err, fmt.Errorf("sum line %d: %w", line.Number, errLine))
		}
		for _, c := range cc {
			s, ok := sums[c]
			if !ok {
				s = &SumLine{Code: c, Name: coder.Describe(c)}
				sums[c] = s
			}
			s.Duration += line.Duration
			for _, t := range line.TaskList {
				if len(t.Description) > 0 {
					s.Description = append(s.Description, t.Description)
				}
				if len(t.Reference) > 0 {
					s.Reference = append(s.Reference, t.Reference)
				}
			}
		}
	}
	list := make([]*SumLine, 0, len(sums))
	for _, item := range sums {
		item.Description = sortDeDuplicate(item.Description)
		item.Reference = sortDeDuplicate(item.Reference)
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Code.Type == b.Code.Type {
			return a.Code.Value < b.Code.Value
		}
		return a.Code.Type < b.Code.Type
	})
	return list, err
}

type stopType int

const (
	ensureStop stopType = iota
	ensureNoStop
	ignoreStop
)

func splitDescription(s string, stop string, st stopType) []Task {
	dd := strings.SplitAfter(s, stop)
	list := make([]Task, 0, len(dd))
	for _, d := range dd {
		d = strings.TrimSpace(d)
		if len(d) == 0 {
			continue
		}
		var code string
		if strings.HasPrefix(d, "[") {
			xf := strings.Index(d, "]")
			code = d[1:xf]
			d = strings.TrimSpace(d[xf+1:])
		}
		if len(d) > 0 {
			switch st {
			default:
				panic("unknown stop type")
			case ensureStop:
				if !strings.HasSuffix(d, stop) {
					d = d + stop
				}
			case ensureNoStop:
				d = strings.TrimRight(d, stop)
			case ignoreStop:
				// Nothing.
			}
		}
		list = append(list, Task{
			Reference:   code,
			Description: d,
		})
	}
	return list
}

func sortDeDuplicate(ss []string) []string {
	sort.Strings(ss)
	wi := 0
	prev := ""
	for _, s := range ss {
		if s == prev {
			continue
		}
		prev = s
		ss[wi] = s
		wi++
	}
	return ss[:wi]
}
