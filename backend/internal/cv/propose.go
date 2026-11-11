package cv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"outreach/internal/ai"
	"outreach/internal/evidence"
)

type JSONCompleter interface {
	CompleteJSON(ctx context.Context, req ai.Request, schema json.RawMessage) (ai.JSONResult, error)
}

func Propose(ctx context.Context, w JSONCompleter, base Spec, jd string, cited []evidence.Item) (Spec, bool, error) {
	if w == nil {
		return base, false, nil
	}
	spec, skip, err := proposeOnce(ctx, w, base, jd, cited)
	if skip {
		return base, false, nil
	}
	if err == nil {
		return spec, false, nil
	}
	spec, skip, err = proposeOnce(ctx, w, base, jd, cited)
	if skip {
		return base, true, nil
	}
	if err == nil {
		return spec, true, nil
	}
	return Spec{}, true, err
}

func proposeOnce(ctx context.Context, w JSONCompleter, base Spec, jd string, cited []evidence.Item) (Spec, bool, error) {
	raw, err := json.Marshal(base)
	if err != nil {
		return Spec{}, false, err
	}
	var ev strings.Builder
	for _, it := range cited {
		fmt.Fprintf(&ev, "- %s\n", it.Text)
	}
	got, err := w.CompleteJSON(ctx, ai.Request{
		Role:        ai.RoleAnalyze,
		Temperature: ai.Temperature(ai.RoleAnalyze),
		System:      "Return the CV JSON. Reorder or drop bullets that match evidence. Do not add employers, dates, or metrics.",
		User:        "CV:\n" + string(raw) + "\nJD:\n" + jd + "\nEvidence:\n" + ev.String(),
	}, json.RawMessage(`{"type":"object"}`))
	if err != nil {
		return Spec{}, false, err
	}
	var spec Spec
	if err := json.Unmarshal(got.Raw, &spec); err != nil || strings.TrimSpace(spec.Name) == "" {
		return Spec{}, true, nil
	}
	if err := CheckSubset(base, spec); err != nil {
		return Spec{}, false, err
	}
	if err := CheckClaims(spec, cited, jd); err != nil {
		return Spec{}, false, err
	}
	return spec, false, nil
}
