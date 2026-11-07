package evidence

import (
	"fmt"
	"regexp"
	"strings"
)

var digits = regexp.MustCompile(`\d+`)

func GateNumbers(body string, cited []Item) error {
	blob := make([]byte, 0, 256)
	for _, it := range cited {
		blob = append(blob, it.Text...)
		blob = append(blob, '\n')
	}
	hay := string(blob)
	for _, n := range digits.FindAllString(body, -1) {
		if !strings.Contains(hay, n) {
			return fmt.Errorf("%w: %s", ErrClaim, n)
		}
	}
	return nil
}
