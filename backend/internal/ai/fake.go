package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
)

type Fake struct {
	Model     string
	Text      string
	JSON      json.RawMessage
	Vectors   [][]float32
	Dim       int
	CompleteN int
	JSONN     int
	EmbedN    int
}

func (f *Fake) Complete(context.Context, Request) (TextResult, error) {
	f.CompleteN++
	model := f.Model
	if model == "" {
		model = "fake"
	}
	return TextResult{Text: f.Text, Model: model}, nil
}

func (f *Fake) CompleteJSON(context.Context, Request, json.RawMessage) (JSONResult, error) {
	f.JSONN++
	model := f.Model
	if model == "" {
		model = "fake"
	}
	raw := f.JSON
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	return JSONResult{Raw: raw, Model: model}, nil
}

func (f *Fake) Embed(_ context.Context, texts []string) ([][]float32, error) {
	f.EmbedN++
	if f.Vectors != nil {
		if len(f.Vectors) != len(texts) {
			return nil, fmt.Errorf("ai: fake embed len %d want %d", len(f.Vectors), len(texts))
		}
		return f.Vectors, nil
	}
	out := make([][]float32, len(texts))
	dim := f.Dim
	if dim <= 0 {
		dim = 1
	}
	for i := range texts {
		if dim == 1 {
			out[i] = []float32{float32(len(texts[i]))}
			continue
		}
		out[i] = hashVec(texts[i], dim)
	}
	return out, nil
}

func hashVec(s string, dim int) []float32 {
	v := make([]float32, dim)
	for i := 0; i < len(s); i++ {
		v[i%dim] += float32(s[i]) / 255
	}
	var n float32
	for _, x := range v {
		n += x * x
	}
	if n == 0 {
		return v
	}
	inv := 1 / float32(math.Sqrt(float64(n)))
	for i := range v {
		v[i] *= inv
	}
	return v
}
