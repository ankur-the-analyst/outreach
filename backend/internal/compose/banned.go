package compose

import (
	_ "embed"
	"strings"
)

//go:embed banned.txt
var bannedFile string

var bannedPhrases = parseBanned(bannedFile)

func parseBanned(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
