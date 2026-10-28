package deck

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	pdfHeader       = "%PDF-1.4\n%\xE2\xE3\xCF\xD3\n"
	firstPageObject = 6
	capHeight       = 0.36
	hairlineWidth   = 0.5
	regularFont     = "/F1"
	boldFont        = "/F2"
)

type pdfWriter struct {
	buf     bytes.Buffer
	offsets []int
}

// WritePDF writes d as a PDF 1.4 document: one page per laid-out sheet, the standard Helvetica fonts.
func WritePDF(w io.Writer, d Deck) error {
	pages := d.sheets()
	doc := &pdfWriter{}
	doc.buf.WriteString(pdfHeader)
	kids := make([]string, len(pages))
	for i := range pages {
		kids[i] = strconv.Itoa(firstPageObject+2*i) + " 0 R"
	}
	doc.object([]byte("<< /Type /Catalog /Pages 2 0 R >>"))
	doc.object(fmt.Appendf(nil, "<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	doc.object([]byte(fontObject("Helvetica")))
	doc.object([]byte(fontObject("Helvetica-Bold")))
	doc.object(infoObject(d))
	for i, p := range pages {
		content := p.pdfContent()
		doc.object(fmt.Appendf(nil, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << %s 3 0 R %s 4 0 R >> >> /Contents %d 0 R >>",
			pageWidth, pageHeight, regularFont, boldFont, firstPageObject+2*i+1))
		stream := fmt.Appendf(nil, "<< /Length %d >>\nstream\n", len(content))
		doc.object(append(append(stream, content...), "\nendstream"...))
	}
	doc.trailer()
	_, err := w.Write(doc.buf.Bytes())
	return err
}

func (p *pdfWriter) object(body []byte) {
	p.offsets = append(p.offsets, p.buf.Len())
	fmt.Fprintf(&p.buf, "%d 0 obj\n", len(p.offsets))
	p.buf.Write(body)
	p.buf.WriteString("\nendobj\n")
}

func (p *pdfWriter) trailer() {
	at := p.buf.Len()
	fmt.Fprintf(&p.buf, "xref\n0 %d\n0000000000 65535 f \n", len(p.offsets)+1)
	for _, offset := range p.offsets {
		fmt.Fprintf(&p.buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&p.buf, "trailer\n<< /Size %d /Root 1 0 R /Info 5 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(p.offsets)+1, at)
}

func fontObject(name string) string {
	return "<< /Type /Font /Subtype /Type1 /BaseFont /" + name + " /Encoding /WinAnsiEncoding >>"
}

func infoObject(d Deck) []byte {
	created := d.Created.UTC().Format("20060102150405")
	return fmt.Appendf(nil, "<< /Title %s /Author %s /Producer %s /CreationDate (D:%sZ) >>",
		unicodeString(d.Title), unicodeString(d.Author), unicodeString(d.Author), created)
}

// unicodeString writes s as a PDF text string in UTF-16 with its byte-order mark, so any title keeps its characters.
func unicodeString(s string) string {
	var b strings.Builder
	b.WriteString("<FEFF")
	for _, unit := range utf16.Encode([]rune(s)) {
		fmt.Fprintf(&b, "%04X", unit)
	}
	b.WriteString(">")
	return b.String()
}

func (s sheet) pdfContent() []byte {
	var b bytes.Buffer
	for _, x := range s.boxes {
		x.pdf(&b)
	}
	for _, g := range s.grids {
		g.pdf(&b)
	}
	for _, t := range s.texts {
		t.pdf(&b)
	}
	return b.Bytes()
}

func (x box) pdf(b *bytes.Buffer) {
	bottom := pageHeight - x.Y - x.H
	if x.Fill != "" {
		fmt.Fprintf(b, "%s rg %d %d %d %d re f\n", rgb(x.Fill), x.X, bottom, x.W, x.H)
	}
	if x.Line != "" {
		fmt.Fprintf(b, "%s RG %.1f w %d %d %d %d re S\n", rgb(x.Line), hairlineWidth, x.X, bottom, x.W, x.H)
	}
}

func (g grid) pdf(b *bytes.Buffer) {
	for r, row := range g.Rows {
		y := g.Y + r*rowHeight
		x := g.X
		for c, cl := range row {
			width := g.Widths[c]
			if cl.Fill != "" && cl.Fill != White {
				box{X: x, Y: y, W: width, H: rowHeight, Fill: cl.Fill}.pdf(b)
			}
			text{X: x + cellPad, Y: y, W: width - 2*cellPad, H: rowHeight, Size: tableSize, Bold: cl.Bold, Colour: Ink,
				Align: cl.Align, Value: cl.Text}.pdf(b)
			x += width
		}
		line := pageHeight - y - rowHeight
		fmt.Fprintf(b, "%s RG %.1f w %d %d m %d %d l S\n", rgb(Hairline), hairlineWidth, g.X, line, x, line)
	}
}

func (t text) pdf(b *bytes.Buffer) {
	if t.Value == "" {
		return
	}
	x := t.X
	switch t.Align {
	case Right:
		x += t.W - textWidth(t.Value, t.Size, t.Bold)
	case Center:
		x += (t.W - textWidth(t.Value, t.Size, t.Bold)) / 2
	case Left:
	}
	font := regularFont
	if t.Bold {
		font = boldFont
	}
	baseline := float64(pageHeight) - float64(t.Y) - float64(t.H)/2 - capHeight*float64(t.Size)
	fmt.Fprintf(b, "BT %s %d Tf %s rg %d %.2f Td (", font, t.Size, rgb(t.Colour), x, baseline)
	b.Write(escapePDF(winANSI(t.Value)))
	b.WriteString(") Tj ET\n")
}

// escapePDF escapes a literal string's delimiters and writes every byte outside printable ASCII in octal.
func escapePDF(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	for _, c := range raw {
		switch {
		case c == '(' || c == ')' || c == '\\':
			out = append(out, '\\', c)
		case c < ' ' || c > '~':
			out = fmt.Appendf(out, "\\%03o", c)
		default:
			out = append(out, c)
		}
	}
	return out
}

func rgb(c Colour) string {
	value, err := strconv.ParseUint(string(c), 16, 32)
	if err != nil {
		value = 0
	}
	return fmt.Sprintf("%.3f %.3f %.3f", float64(value>>16&0xFF)/255, float64(value>>8&0xFF)/255, float64(value&0xFF)/255)
}
