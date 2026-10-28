package deck

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	emuPerPoint       = 12700
	firstSlideID      = 256
	firstSlideRel     = 3
	coreProperties    = `application/vnd.openxmlformats-package.core-properties+xml`
	themeContentType  = `application/vnd.openxmlformats-officedocument.theme+xml`
	appContentType    = `application/vnd.openxmlformats-officedocument.extended-properties+xml`
	tableStyleContent = `application/vnd.openxmlformats-officedocument.presentationml.tableStyles+xml`
	relsClose         = `</Relationships>`
	w3cTime           = "2006-01-02T15:04:05Z"
)

type part struct {
	name, body string
}

// WritePPTX writes d as a 16:9 PowerPoint presentation: one slide per laid-out sheet.
func WritePPTX(w io.Writer, d Deck) error {
	sheets := d.sheets()
	parts := []part{
		{"[Content_Types].xml", contentTypes(len(sheets))},
		{"_rels/.rels", packageRels},
		{"docProps/core.xml", coreXML(d)},
		{"docProps/app.xml", appXML(d.Author, len(sheets))},
		{"ppt/presentation.xml", presentationXML(len(sheets))},
		{"ppt/_rels/presentation.xml.rels", presentationRels(len(sheets))},
		{"ppt/presProps.xml", presProps},
		{"ppt/viewProps.xml", viewProps},
		{"ppt/tableStyles.xml", tableStyles},
		{"ppt/theme/theme1.xml", theme},
		{"ppt/slideMasters/slideMaster1.xml", slideMaster},
		{"ppt/slideMasters/_rels/slideMaster1.xml.rels", slideMasterRels},
		{"ppt/slideLayouts/slideLayout1.xml", slideLayout},
		{"ppt/slideLayouts/_rels/slideLayout1.xml.rels", slideLayoutRels},
	}
	for i, s := range sheets {
		parts = append(parts,
			part{fmt.Sprintf("ppt/slides/slide%d.xml", i+1), s.slideXML()},
			part{fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i+1), slideRels})
	}
	return writeZip(w, parts, d.Created)
}

func writeZip(w io.Writer, parts []part, modified time.Time) error {
	zw := zip.NewWriter(w)
	for _, p := range parts {
		f, err := zw.CreateHeader(&zip.FileHeader{Name: p.name, Method: zip.Deflate, Modified: modified.UTC()})
		if err != nil {
			return err
		}
		if _, err := io.WriteString(f, xmlDeclaration+p.body); err != nil {
			return err
		}
	}
	return zw.Close()
}

func contentTypes(slides int) string {
	var b strings.Builder
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>`)
	overrides := [][2]string{
		{"/ppt/presentation.xml", contentType + "presentation.main+xml"},
		{"/ppt/slideMasters/slideMaster1.xml", contentType + "slideMaster+xml"},
		{"/ppt/slideLayouts/slideLayout1.xml", contentType + "slideLayout+xml"},
		{"/ppt/theme/theme1.xml", themeContentType},
		{"/ppt/presProps.xml", contentType + "presProps+xml"},
		{"/ppt/viewProps.xml", contentType + "viewProps+xml"},
		{"/ppt/tableStyles.xml", tableStyleContent},
		{"/docProps/core.xml", coreProperties},
		{"/docProps/app.xml", appContentType},
	}
	for i := range slides {
		overrides = append(overrides, [2]string{fmt.Sprintf("/ppt/slides/slide%d.xml", i+1), contentType + "slide+xml"})
	}
	for _, o := range overrides {
		fmt.Fprintf(&b, `<Override PartName="%s" ContentType="%s"/>`, o[0], o[1])
	}
	b.WriteString(`</Types>`)
	return b.String()
}

func presentationXML(slides int) string {
	var b strings.Builder
	b.WriteString(`<p:presentation ` + pmlNamespaces + ` saveSubsetFonts="1">` +
		`<p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst><p:sldIdLst>`)
	for i := range slides {
		fmt.Fprintf(&b, `<p:sldId id="%d" r:id="rId%d"/>`, firstSlideID+i, firstSlideRel+i)
	}
	fmt.Fprintf(&b, `</p:sldIdLst><p:sldSz cx="%d" cy="%d"/><p:notesSz cx="6858000" cy="9144000"/></p:presentation>`,
		pageWidth*emuPerPoint, pageHeight*emuPerPoint)
	return b.String()
}

func presentationRels(slides int) string {
	var b strings.Builder
	b.WriteString(relsOpen +
		`<Relationship Id="rId1" Type="` + relType + `slideMaster" Target="slideMasters/slideMaster1.xml"/>` +
		`<Relationship Id="rId2" Type="` + relType + `theme" Target="theme/theme1.xml"/>`)
	for i := range slides {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="%sslide" Target="slides/slide%d.xml"/>`, firstSlideRel+i, relType, i+1)
	}
	next := firstSlideRel + slides
	for i, target := range []string{"presProps", "viewProps", "tableStyles"} {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="%s%s" Target="%s.xml"/>`, next+i, relType, target, target)
	}
	b.WriteString(relsClose)
	return b.String()
}

func coreXML(d Deck) string {
	created := d.Created.UTC().Format(w3cTime)
	return `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" ` +
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>` + escapeXML(d.Title) + `</dc:title>` +
		`<dc:creator>` + escapeXML(d.Author) + `</dc:creator>` +
		`<dcterms:created xsi:type="dcterms:W3CDTF">` + created + `</dcterms:created>` +
		`<dcterms:modified xsi:type="dcterms:W3CDTF">` + created + `</dcterms:modified></cp:coreProperties>`
}

func appXML(author string, slides int) string {
	return fmt.Sprintf(`<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties">`+
		`<Application>%s</Application><Slides>%d</Slides></Properties>`, escapeXML(author), slides)
}

func escapeXML(s string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(clean(s))); err != nil {
		return ""
	}
	return b.String()
}
