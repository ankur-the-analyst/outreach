package cv

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Spec struct {
	Name     string    `json:"name"`
	Headline string    `json:"headline,omitempty"`
	Sections []Section `json:"sections,omitempty"`
}

type Section struct {
	Title string   `json:"title"`
	Lines []string `json:"lines,omitempty"`
}

func PDF(spec Spec, from string) ([]byte, error) {
	from = strings.TrimSpace(from)
	if from == "" {
		return nil, errors.New("cv: from required")
	}
	var lines []string
	if n := strings.TrimSpace(spec.Name); n != "" {
		lines = append(lines, n)
	}
	if h := strings.TrimSpace(spec.Headline); h != "" {
		lines = append(lines, h)
	}
	lines = append(lines, from)
	for _, s := range spec.Sections {
		if t := strings.TrimSpace(s.Title); t != "" {
			lines = append(lines, t)
		}
		for _, ln := range s.Lines {
			if t := strings.TrimSpace(ln); t != "" {
				lines = append(lines, t)
			}
		}
	}
	return pdf(lines)
}

func FileName(spec Spec) string {
	base := slug(spec.Name)
	if base == "" {
		base = "cv"
	}
	return base + ".pdf"
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func pdf(lines []string) ([]byte, error) {
	var stream bytes.Buffer
	stream.WriteString("BT /F1 12 Tf 72 720 Td")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString(" 0 -16 Td")
		}
		fmt.Fprintf(&stream, " (%s) Tj", pdfString(line))
	}
	stream.WriteString(" ET")
	body := stream.Bytes()

	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(body), body),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offs := make([]int, len(objs)+1)
	for i, obj := range objs {
		offs[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	startxref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for i := 1; i <= len(objs); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offs[i])
	}
	fmt.Fprintf(&out, "trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, startxref)
	return out.Bytes(), nil
}

func pdfString(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return ' '
		}
		return r
	}, s)
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	return s
}
