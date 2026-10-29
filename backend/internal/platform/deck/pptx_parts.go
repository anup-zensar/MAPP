package deck

// The fixed parts of a presentation: the minimal set PowerPoint's own blank presentation has (ECMA-376 part 1).
const (
	xmlDeclaration = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"
	nsDrawing      = `http://schemas.openxmlformats.org/drawingml/2006/main`
	nsRelations    = `http://schemas.openxmlformats.org/officeDocument/2006/relationships`
	nsPresentation = `http://schemas.openxmlformats.org/presentationml/2006/main`
	pmlNamespaces  = `xmlns:a="` + nsDrawing + `" xmlns:r="` + nsRelations + `" xmlns:p="` + nsPresentation + `"`
	relType        = nsRelations + `/`
	relsOpen       = `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`
	contentType    = `application/vnd.openxmlformats-officedocument.presentationml.`
	emptyGroup     = `<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>` +
		`<p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>`

	packageRels = relsOpen +
		`<Relationship Id="rId1" Type="` + relType + `officeDocument" Target="ppt/presentation.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>` +
		`<Relationship Id="rId3" Type="` + relType + `extended-properties" Target="docProps/app.xml"/>` +
		`</Relationships>`

	slideMaster = `<p:sldMaster ` + pmlNamespaces + `><p:cSld><p:bg><p:bgRef idx="1001"><a:schemeClr val="bg1"/></p:bgRef></p:bg>` +
		`<p:spTree>` + emptyGroup + `</p:spTree></p:cSld>` +
		`<p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" ` +
		`accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>` +
		`<p:sldLayoutIdLst><p:sldLayoutId id="2147483649" r:id="rId1"/></p:sldLayoutIdLst>` +
		`<p:txStyles><p:titleStyle/><p:bodyStyle/><p:otherStyle/></p:txStyles></p:sldMaster>`

	slideMasterRels = relsOpen +
		`<Relationship Id="rId1" Type="` + relType + `slideLayout" Target="../slideLayouts/slideLayout1.xml"/>` +
		`<Relationship Id="rId2" Type="` + relType + `theme" Target="../theme/theme1.xml"/></Relationships>`

	slideLayout = `<p:sldLayout ` + pmlNamespaces + ` type="blank" preserve="1"><p:cSld name="Blank"><p:spTree>` +
		emptyGroup + `</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sldLayout>`

	slideLayoutRels = relsOpen +
		`<Relationship Id="rId1" Type="` + relType + `slideMaster" Target="../slideMasters/slideMaster1.xml"/></Relationships>`

	slideRels = relsOpen +
		`<Relationship Id="rId1" Type="` + relType + `slideLayout" Target="../slideLayouts/slideLayout1.xml"/></Relationships>`

	presProps   = `<p:presentationPr ` + pmlNamespaces + `/>`
	viewProps   = `<p:viewPr ` + pmlNamespaces + `/>`
	tableStyles = `<a:tblStyleLst xmlns:a="` + nsDrawing + `" def="{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}"/>`

	phFill    = `<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>`
	threeFill = phFill + phFill + phFill
	themeFont = `<a:latin typeface="Arial"/><a:ea typeface=""/><a:cs typeface=""/>`
	theme     = `<a:theme xmlns:a="` + nsDrawing + `" name="SP&amp;A One"><a:themeElements>` +
		`<a:clrScheme name="SP&amp;A One"><a:dk1><a:srgbClr val="292929"/></a:dk1><a:lt1><a:srgbClr val="FFFFFF"/></a:lt1>` +
		`<a:dk2><a:srgbClr val="003C88"/></a:dk2><a:lt2><a:srgbClr val="F5F5F5"/></a:lt2>` +
		`<a:accent1><a:srgbClr val="336094"/></a:accent1><a:accent2><a:srgbClr val="008557"/></a:accent2>` +
		`<a:accent3><a:srgbClr val="008192"/></a:accent3><a:accent4><a:srgbClr val="5B6B7A"/></a:accent4>` +
		`<a:accent5><a:srgbClr val="87217D"/></a:accent5><a:accent6><a:srgbClr val="C77700"/></a:accent6>` +
		`<a:hlink><a:srgbClr val="4B78AA"/></a:hlink><a:folHlink><a:srgbClr val="9A60A4"/></a:folHlink></a:clrScheme>` +
		`<a:fontScheme name="SP&amp;A One"><a:majorFont>` + themeFont + `</a:majorFont><a:minorFont>` + themeFont +
		`</a:minorFont></a:fontScheme><a:fmtScheme name="SP&amp;A One"><a:fillStyleLst>` + threeFill + `</a:fillStyleLst>` +
		`<a:lnStyleLst><a:ln w="6350">` + phFill + `</a:ln><a:ln w="12700">` + phFill + `</a:ln><a:ln w="19050">` + phFill +
		`</a:ln></a:lnStyleLst><a:effectStyleLst><a:effectStyle><a:effectLst/></a:effectStyle><a:effectStyle><a:effectLst/>` +
		`</a:effectStyle><a:effectStyle><a:effectLst/></a:effectStyle></a:effectStyleLst>` +
		`<a:bgFillStyleLst>` + threeFill + `</a:bgFillStyleLst></a:fmtScheme></a:themeElements>` +
		`<a:objectDefaults/><a:extraClrSchemeLst/></a:theme>`
)
