package datatable

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Type int

const (
	TypeCSV Type = 1
	TypeTSV Type = 2
)

type Writer interface {
	Line(values ...any) error
	Close() error
}

type tsv struct {
	tw *tabwriter.Writer
}
type csv struct {
	buf *bufio.Writer
}

func (w tsv) Line(values ...any) error {
	for i, v := range values {
		if i > 0 {
			fmt.Fprint(w.tw, "\t")
		}
		fmt.Fprint(w.tw, encodeValue(v))
	}
	fmt.Fprint(w.tw, "\n")
	return nil
}
func (w tsv) Close() error {
	return w.tw.Flush()
}

var csvReplacer = strings.NewReplacer(`"`, `""`)

func (csv) cell(w io.Writer, v any) error {
	s := encodeValue(v)
	var err error
	for _, r := range s {
		switch r {
		case '"', ',':
			_, err = fmt.Fprint(w, `"`)
			if err != nil {
				return err
			}
			_, err = csvReplacer.WriteString(w, s)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(w, `"`)
			if err != nil {
				return err
			}
			return nil
		}
	}
	_, err = fmt.Fprint(w, s)
	return err
}

func (w csv) Line(values ...any) error {
	var err error
	for i, v := range values {
		if i > 0 {
			fmt.Fprint(w.buf, ",")
		}
		err = w.cell(w.buf, v)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w.buf, "\n")
	return nil
}
func (w csv) Close() error {
	return w.buf.Flush()
}

func NewTSV(w io.Writer) Writer {
	return tsv{
		tw: tabwriter.NewWriter(w, 2, 8, 2, '\t', 0),
	}
}
func NewCSV(w io.Writer) Writer {
	return csv{
		buf: bufio.NewWriter(w),
	}
}

func encodeValue(v any) string {
	switch v := v.(type) {
	default:
		return fmt.Sprint(v)
	case string:
		return v
	}
}
