package compose

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"outreach/internal/evidence"
)

const (
	MaxWords = 125
	MaxLinks = 1
)

type Fact struct {
	Claim      string    `json:"claim"`
	EvidenceID uuid.UUID `json:"evidence_id"`
}

type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	linkRe = regexp.MustCompile(`(?i)https?://[^\s]+`)
)

func Leftover(parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(p, "{{") {
			return true
		}
	}
	return false
}

func Check(body string, facts []Fact, cited []evidence.Item) []Failure {
	var out []Failure
	if Leftover(body) {
		out = append(out, Failure{Code: "template", Message: "leftover {{ in body"})
	}
	if n := wordCount(body); n > MaxWords {
		out = append(out, Failure{Code: "length", Message: fmt.Sprintf("%d words; max %d", n, MaxWords)})
	}
	if n := len(linkRe.FindAllString(body, -1)); n > MaxLinks {
		out = append(out, Failure{Code: "links", Message: fmt.Sprintf("%d links; max %d", n, MaxLinks)})
	}
	lower := strings.ToLower(body)
	for _, phrase := range bannedPhrases {
		if phrase != "" && strings.Contains(lower, phrase) {
			out = append(out, Failure{Code: "banned", Message: phrase})
		}
	}
	allowed := make(map[uuid.UUID]struct{}, len(cited))
	for _, it := range cited {
		allowed[it.ID] = struct{}{}
	}
	for _, f := range facts {
		if _, ok := allowed[f.EvidenceID]; !ok {
			out = append(out, Failure{Code: "uncited", Message: f.EvidenceID.String()})
		}
	}
	if err := evidence.GateNumbers(body, cited); err != nil {
		out = append(out, Failure{Code: "claim", Message: err.Error()})
	}
	return out
}

func wordCount(s string) int {
	n := 0
	in := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			in = false
			continue
		}
		if !in {
			n++
			in = true
		}
	}
	return n
}
