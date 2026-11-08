package jobsjd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/ai"
)

func TestJSONWriterFallsBackWithoutBinding(t *testing.T) {
	t.Parallel()
	fb := &ai.Fake{JSON: json.RawMessage(`{"summary":"ok"}`)}
	p := NewPostgres(nil, nil, fb).WithBinder(stubBind{err: ai.ErrNoBinding})
	w, err := p.jsonWriter(context.Background(), uuid.Nil)
	if err != nil || w != fb {
		t.Fatalf("%T %v", w, err)
	}
}

type stubBind struct {
	p   ai.Provider
	err error
}

func (s stubBind) Provider(context.Context, uuid.UUID, ai.Role) (ai.Provider, error) {
	return s.p, s.err
}

type errJSON struct{}

func (errJSON) CompleteJSON(context.Context, ai.Request, json.RawMessage) (ai.JSONResult, error) {
	return ai.JSONResult{}, errors.New("upstream")
}

func TestScoreFitKeepsRowWhenModelFails(t *testing.T) {
	t.Parallel()
	score, raw := scoreFit(context.Background(), errJSON{}, "job", "resume")
	if score.Valid || string(raw) != `{"score":0,"summary":""}` {
		t.Fatalf("%v %s", score, raw)
	}
	fb := &ai.Fake{JSON: json.RawMessage(`{"score":40,"summary":"overlap"}`)}
	score, raw = scoreFit(context.Background(), fb, "job", "resume")
	if !score.Valid || score.Int32 != 40 || string(raw) != `{"score":40,"summary":"overlap"}` {
		t.Fatalf("%v %s", score, raw)
	}
}
