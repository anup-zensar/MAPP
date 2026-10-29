package deck

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestWritePPTX_APresentationOfOneSlidePerPage_036_FR023(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := WritePPTX(&buf, sampleDeck(40)); err != nil {
		t.Fatalf("WritePPTX: %v", err)
	}
	files := unzip(t, buf.Bytes())
	checkParts(t, files)
	if slides := strings.Count(files["ppt/presentation.xml"], "<p:sldId "); slides != 4 {
		t.Errorf("slides = %d, want 4", slides)
	}
	first, second := files["ppt/slides/slide1.xml"], files["ppt/slides/slide2.xml"]
	if !strings.Contains(first, "FY2026 R&amp;D (Q1) – 中国") || !strings.Contains(first, `b="1"`) {
		t.Errorf("slide 1 does not hold the escaped title in bold")
	}
	if !strings.Contains(second, "<a:tbl>") || !strings.Contains(second, `<a:srgbClr val="008557"/>`) {
		t.Errorf("slide 2 has no table or no actual bar")
	}
}

func checkParts(t *testing.T, files map[string]string) {
	t.Helper()
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "ppt/presentation.xml", "ppt/theme/theme1.xml",
		"ppt/slideMasters/slideMaster1.xml", "ppt/slideLayouts/slideLayout1.xml", "ppt/slides/slide4.xml", "docProps/core.xml"} {
		if _, found := files[name]; !found {
			t.Errorf("part %s is missing", name)
		}
	}
	for name, body := range files {
		if err := wellFormed(body); err != nil {
			t.Errorf("%s is not well-formed XML: %v", name, err)
		}
	}
}

func TestWritePDF_ADocumentOfTheSamePages_036_FR023(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := WritePDF(&buf, sampleDeck(40)); err != nil {
		t.Fatalf("WritePDF: %v", err)
	}
	pdf := buf.String()

	if !strings.HasPrefix(pdf, "%PDF-1.4\n") || !strings.HasSuffix(pdf, "%%EOF\n") {
		t.Fatalf("not a PDF file: %q … %q", pdf[:10], pdf[len(pdf)-6:])
	}
	if pages := len(regexp.MustCompile(`/Type /Page `).FindAllString(pdf, -1)); pages != 4 {
		t.Errorf("pages = %d, want 4", pages)
	}
	if !strings.Contains(pdf, `(FY2026 R&D \(Q1\) \226 ??) Tj`) {
		t.Errorf("the title is not escaped and encoded for WinAnsi")
	}
	checkXref(t, pdf)
}

func checkXref(t *testing.T, pdf string) {
	t.Helper()
	at, err := strconv.Atoi(regexp.MustCompile(`startxref\n(\d+)\n`).FindStringSubmatch(pdf)[1])
	if err != nil || !strings.HasPrefix(pdf[at:], "xref\n") {
		t.Fatalf("startxref does not point at the xref table")
	}
	entries := regexp.MustCompile(`(\d{10}) 00000 n `).FindAllStringSubmatch(pdf[at:], -1)
	for i, e := range entries {
		offset, _ := strconv.Atoi(e[1])
		if want := strconv.Itoa(i+1) + " 0 obj\n"; !strings.HasPrefix(pdf[offset:], want) {
			t.Errorf("xref entry %d points at %q, want %q", i+1, pdf[offset:offset+10], want)
		}
	}
}

func unzip(t *testing.T, data []byte) map[string]string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a zip file: %v", err)
	}
	files := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		files[f.Name] = string(body)
	}
	return files
}

func wellFormed(body string) error {
	d := xml.NewDecoder(strings.NewReader(body))
	for {
		if _, err := d.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
