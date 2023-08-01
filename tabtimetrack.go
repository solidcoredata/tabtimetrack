package tabtimetrack

import (
	"bytes"
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
	Title string
	List  []Line
}

type Line struct {
	Date        civil.Date
	Start       civil.Time // Inclusive.
	Stop        civil.Time // Inclusive.
	Duration    time.Duration
	Description string
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
	ll := bytes.Split(data, []byte{'\n'})
	for i, l := range ll {
		ln := i + 1
		ww := bytes.Split(l, []byte{'\t'})
		if i == 0 && len(ww) == 1 {
			f.Title = string(ww[0])
			continue
		}
		if len(ww) < 3 {
			if len(ww[0]) == 0 {
				continue
			}
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
			Date:        d,
			Start:       ts,
			Stop:        te,
			Duration:    dur,
			Description: desc,
		})
	}
	return f, err
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

func SumFunc(lineList []Line, f func(d civil.Date) []Code, desc func(code Code) string) []*SumLine {
	sums := make(map[Code]*SumLine, 100)
	for _, line := range lineList {
		cc := f(line.Date)
		for _, c := range cc {
			s, ok := sums[c]
			if !ok {
				s = &SumLine{Code: c, Name: desc(c)}
				sums[c] = s
			}
			s.Duration += line.Duration
			if len(line.Description) > 0 {
				s.Description = append(s.Description, line.Description)
			}
		}
	}
	list := make([]*SumLine, 0, len(sums))
	for _, item := range sums {
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Code.Type == b.Code.Type {
			return a.Code.Value < b.Code.Value
		}
		return a.Code.Type < b.Code.Type
	})
	return list
}

func SplitSortDeDuplicate(ss []string, stop string) []string {
	return sortDeDuplicate(splitAppend(ss, stop))
}

func splitAppend(ss []string, stop string) []string {
	descList := make([]string, 0, len(ss))
	for _, s := range ss {
		dd := strings.SplitAfter(s, stop)
		for _, d := range dd {
			d = strings.TrimSpace(d)
			if len(d) == 0 {
				continue
			}

			if !strings.HasSuffix(d, stop) {
				d = d + stop
			}
			descList = append(descList, d)
		}
	}
	return descList
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
