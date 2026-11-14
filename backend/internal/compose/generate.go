package compose

import (
	"context"
	"fmt"
	"strings"

	"outreach/internal/ai"
	"outreach/internal/evidence"
)

type Completer interface {
	Complete(ctx context.Context, req ai.Request) (ai.TextResult, error)
}

func Generate(ctx context.Context, w Completer, cited []evidence.Item, facts []Fact, about string) (body string, regenerated bool, failures []Failure, err error) {
	if w == nil {
		return "", false, nil, fmt.Errorf("compose: writer required")
	}
	req := ai.Request{
		Role:        ai.RoleCompose,
		Temperature: ai.Temperature(ai.RoleCompose),
		System:      "Write a short first-touch email. Cite only provided evidence. You may use the notes about this person. No leftover {{ tokens. At most one link. At most 125 words.",
		User:        evidencePrompt(cited, facts, about),
	}
	got, err := w.Complete(ctx, req)
	if err != nil {
		return "", false, nil, err
	}
	body = strings.TrimSpace(got.Text)
	failures = Check(body, facts, cited, about)
	if len(failures) == 0 {
		return body, false, nil, nil
	}
	got, err = w.Complete(ctx, req)
	if err != nil {
		return "", true, failures, err
	}
	body = strings.TrimSpace(got.Text)
	return body, true, Check(body, facts, cited, about), nil
}

func evidencePrompt(cited []evidence.Item, facts []Fact, about string) string {
	var b strings.Builder
	if note := strings.TrimSpace(about); note != "" {
		b.WriteString("About this person:\n")
		b.WriteString(note)
		b.WriteString("\n")
	}
	b.WriteString("Evidence:\n")
	for _, it := range cited {
		fmt.Fprintf(&b, "- [%s] %s\n", it.ID, it.Text)
	}
	b.WriteString("Facts to use:\n")
	for _, f := range facts {
		fmt.Fprintf(&b, "- %s (%s)\n", f.Claim, f.EvidenceID)
	}
	return b.String()
}
