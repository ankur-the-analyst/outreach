package mime

import (
	"html"
	"strings"
)

const OptOutLine = "If you'd rather not get these emails, reply and say so."

func CommercialFooter(legalName, reply, postal, unsubURL string) string {
	var b strings.Builder
	b.WriteString("--\n")
	if s := strings.TrimSpace(legalName); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	if s := strings.TrimSpace(reply); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	if s := strings.TrimSpace(postal); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	if s := strings.TrimSpace(unsubURL); s != "" {
		b.WriteString("Unsubscribe: ")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func WithFooter(text, htmlBody, footer string) (string, string) {
	footer = strings.TrimSpace(footer)
	if footer == "" {
		return text, htmlBody
	}
	text = strings.TrimRight(text, "\n") + "\n\n" + footer + "\n"
	if strings.TrimSpace(htmlBody) == "" {
		return text, htmlBody
	}
	escaped := html.EscapeString(footer)
	escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
	htmlBody = strings.TrimRight(htmlBody, "\n") + "\n<p>" + escaped + "</p>\n"
	return text, htmlBody
}
