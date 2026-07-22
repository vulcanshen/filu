package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ledongthuc/pdf"
)

const (
	pdfMaxPages = 20
	pdfMaxBytes = 128 * 1024
)

// pdfText extracts plain text from the first pdfMaxPages pages (bounded by
// pdfMaxBytes) plus the total page count. ok is false when the file can't be
// read. The extractor is wrapped in recover() because the library panics on
// some malformed PDFs — a bad file must degrade to the hex fallback, not crash.
func pdfText(path string) (text string, pages int, ok bool) {
	defer func() {
		if recover() != nil {
			text, ok = "", false
		}
	}()
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", 0, false
	}
	defer f.Close()

	pages = r.NumPage()
	fonts := make(map[string]*pdf.Font) // reused so shared fonts resolve once
	var b strings.Builder
	for i := 1; i <= pages && i <= pdfMaxPages; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		s, err := p.GetPlainText(fonts)
		if err != nil {
			continue
		}
		b.WriteString(s)
		if b.Len() >= pdfMaxBytes {
			break
		}
	}
	return b.String(), pages, true
}

// pdfLines formats extracted PDF text for the preview: a dim page-count header
// then the sanitised body (or a note when there's no extractable text layer).
func pdfLines(text string, pages int) []string {
	unit := "pages"
	if pages == 1 {
		unit = "page"
	}
	dim := lipgloss.NewStyle().Foreground(dimColor)
	head := dim.Render(fmt.Sprintf("%d %s", pages, unit))
	if strings.TrimSpace(text) == "" {
		return []string{head, "", dim.Render("(no extractable text)")}
	}
	return append([]string{head, ""}, sanitizeLines(strings.Split(text, "\n"))...)
}
