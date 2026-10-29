package deck

import (
	"fmt"
	"strings"
)

const (
	cellInset    = cellPad * emuPerPoint
	hairlineEMU  = 6350
	runOpen      = `<a:r><a:rPr lang="en-GB" sz="%d"%s dirty="0"><a:solidFill><a:srgbClr val="%s"/></a:solidFill>`
	runFont      = `<a:latin typeface="Arial"/><a:cs typeface="Arial"/></a:rPr><a:t>%s</a:t></a:r>`
	emptyRun     = `<a:endParaRPr lang="en-GB" sz="%d" dirty="0"/>`
	noLine       = `<a:ln><a:noFill/></a:ln>`
	xfrm         = `<a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>`
	rectGeometry = `<a:prstGeom prst="rect"><a:avLst/></a:prstGeom>`
)

type slideWriter struct {
	b    strings.Builder
	next int
}

func (s sheet) slideXML() string {
	w := &slideWriter{next: 2}
	w.b.WriteString(`<p:sld ` + pmlNamespaces + `><p:cSld><p:spTree>` + emptyGroup)
	for _, x := range s.boxes {
		w.box(x)
	}
	for _, g := range s.grids {
		w.table(g)
	}
	for _, t := range s.texts {
		w.text(t)
	}
	w.b.WriteString(`</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sld>`)
	return w.b.String()
}

func (w *slideWriter) id() int {
	w.next++
	return w.next - 1
}

func (w *slideWriter) box(x box) {
	fmt.Fprintf(&w.b, `<p:sp><p:nvSpPr><p:cNvPr id="%d" name="Shape %[1]d"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr><p:spPr>`, w.id())
	fmt.Fprintf(&w.b, xfrm, emu(x.X), emu(x.Y), emu(x.W), emu(x.H))
	w.b.WriteString(rectGeometry + fill(x.Fill))
	if x.Line == "" {
		w.b.WriteString(noLine)
	} else {
		fmt.Fprintf(&w.b, `<a:ln w="%d">%s</a:ln>`, hairlineEMU, fill(x.Line))
	}
	w.b.WriteString(`</p:spPr></p:sp>`)
}

func (w *slideWriter) text(t text) {
	if t.Value == "" {
		return
	}
	fmt.Fprintf(&w.b, `<p:sp><p:nvSpPr><p:cNvPr id="%d" name="Text %[1]d"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr><p:spPr>`, w.id())
	fmt.Fprintf(&w.b, xfrm, emu(t.X), emu(t.Y), emu(t.W), emu(t.H))
	w.b.WriteString(rectGeometry + `<a:noFill/></p:spPr><p:txBody><a:bodyPr wrap="none" lIns="0" tIns="0" rIns="0" bIns="0" ` +
		`anchor="ctr"><a:noAutofit/></a:bodyPr><a:lstStyle/>`)
	w.paragraph(t)
	w.b.WriteString(`</p:txBody></p:sp>`)
}

func (w *slideWriter) paragraph(t text) {
	fmt.Fprintf(&w.b, `<a:p><a:pPr algn="%s"/>`, alignment(t.Align))
	if t.Value == "" {
		fmt.Fprintf(&w.b, emptyRun, t.Size*100)
	} else {
		weight := ""
		if t.Bold {
			weight = ` b="1"`
		}
		fmt.Fprintf(&w.b, runOpen+runFont, t.Size*100, weight, t.Colour, escapeXML(t.Value))
	}
	w.b.WriteString(`</a:p>`)
}

func (w *slideWriter) table(g grid) {
	width := 0
	for _, column := range g.Widths {
		width += column
	}
	fmt.Fprintf(&w.b, `<p:graphicFrame><p:nvGraphicFramePr><p:cNvPr id="%d" name="Table %[1]d"/><p:cNvGraphicFramePr>`+
		`<a:graphicFrameLocks noGrp="1"/></p:cNvGraphicFramePr><p:nvPr/></p:nvGraphicFramePr>`, w.id())
	fmt.Fprintf(&w.b, `<p:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></p:xfrm>`, emu(g.X), emu(g.Y), emu(width), emu(g.height()))
	w.b.WriteString(`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table"><a:tbl>` +
		`<a:tblPr firstRow="1"/><a:tblGrid>`)
	for _, column := range g.Widths {
		fmt.Fprintf(&w.b, `<a:gridCol w="%d"/>`, emu(column))
	}
	w.b.WriteString(`</a:tblGrid>`)
	for _, row := range g.Rows {
		fmt.Fprintf(&w.b, `<a:tr h="%d">`, emu(rowHeight))
		for _, c := range row {
			w.tableCell(c)
		}
		w.b.WriteString(`</a:tr>`)
	}
	w.b.WriteString(`</a:tbl></a:graphicData></a:graphic></p:graphicFrame>`)
}

func (w *slideWriter) tableCell(c cell) {
	w.b.WriteString(`<a:tc><a:txBody><a:bodyPr/><a:lstStyle/>`)
	w.paragraph(text{Value: c.Text, Size: tableSize, Bold: c.Bold, Colour: Ink, Align: c.Align})
	fmt.Fprintf(&w.b, `</a:txBody><a:tcPr marL="%d" marR="%[1]d" marT="0" marB="0" anchor="ctr">`, cellInset)
	for _, side := range []string{"lnL", "lnR", "lnT"} {
		fmt.Fprintf(&w.b, `<a:%s w="0"><a:noFill/></a:%[1]s>`, side)
	}
	fmt.Fprintf(&w.b, `<a:lnB w="%d">%s</a:lnB>%s</a:tcPr></a:tc>`, hairlineEMU, fill(Hairline), fill(c.Fill))
}

func fill(c Colour) string {
	if c == "" {
		return `<a:noFill/>`
	}
	return `<a:solidFill><a:srgbClr val="` + string(c) + `"/></a:solidFill>`
}

func alignment(a Align) string {
	switch a {
	case Right:
		return "r"
	case Center:
		return "ctr"
	case Left:
	}
	return "l"
}

func emu(points int) int {
	return points * emuPerPoint
}
