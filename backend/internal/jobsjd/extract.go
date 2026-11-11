package jobsjd

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

var (
	ErrFile = errors.New("upload a PDF or paste the resume")
	ErrRead = errors.New("could not read this file")
)

var xmlTag = regexp.MustCompile(`<[^>]+>`)

func Extract(mime, name string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrFile
	}
	kind := strings.ToLower(strings.TrimSpace(mime))
	if kind == "" {
		kind = sniff(name, data)
	}
	switch {
	case strings.Contains(kind, "pdf") || strings.HasSuffix(strings.ToLower(name), ".pdf"):
		return pdfText(data)
	case strings.Contains(kind, "wordprocessingml") || strings.HasSuffix(strings.ToLower(name), ".docx"):
		return docxText(data)
	case strings.HasPrefix(kind, "text/") || strings.HasSuffix(strings.ToLower(name), ".txt"):
		t := strings.TrimSpace(string(data))
		if t == "" {
			return "", ErrRead
		}
		return t, nil
	default:
		if t := strings.TrimSpace(string(data)); looksText(t) {
			return t, nil
		}
		return "", ErrRead
	}
}

func sniff(name string, data []byte) string {
	if len(data) >= 4 && string(data[:4]) == "%PDF" {
		return "application/pdf"
	}
	if strings.HasSuffix(strings.ToLower(name), ".docx") {
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	return "application/octet-stream"
}

func pdfText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", ErrRead
	}
	b, err := r.GetPlainText()
	if err != nil {
		return "", ErrRead
	}
	raw, err := io.ReadAll(b)
	if err != nil {
		return "", ErrRead
	}
	t := strings.TrimSpace(string(raw))
	if t == "" {
		return "", ErrRead
	}
	return t, nil
}

func docxText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", ErrRead
	}
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", ErrRead
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return "", ErrRead
		}
		t := strings.TrimSpace(xmlTag.ReplaceAllString(string(raw), " "))
		t = strings.Join(strings.Fields(t), " ")
		if t == "" {
			return "", ErrRead
		}
		return t, nil
	}
	return "", ErrRead
}

func looksText(s string) bool {
	if s == "" {
		return false
	}
	n := 0
	for _, r := range s {
		if r == 0 {
			return false
		}
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			n++
		}
	}
	return n > len(s)/2
}
