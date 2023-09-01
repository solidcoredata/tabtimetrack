package tabtimetrack

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	list := []struct {
		Name  string
		Input string
		Error string
	}{
		{
			Name: "overlap",
			Input: `overlap
2023-08-01	08:00	09:00
2023-08-01	09:00	10:00
`,
			Error: `line 2 overlaps line 3, ensure start and stop are not the same`,
		},
		{
			Name: "offset-min",
			Input: `overlap
2023-08-01	08:00	09:00
2023-08-01	09:01	10:00
`,
			Error: ``,
		},
		{
			Name: "offset-sec",
			Input: `overlap
2023-08-01	08:00	09:00:34
2023-08-01	09:00:35	10:00
`,
			Error: ``,
		},
	}
	for _, item := range list {
		t.Run(item.Name, func(t *testing.T) {
			f, err := Parse([]byte(item.Input))
			var errs string
			if err != nil {
				errs = strings.TrimSpace(err.Error())
			}
			if g, w := errs, item.Error; g != w {
				t.Fatalf("got %s\nwant: %s", g, w)
			}
			_ = f
		})
	}
}
