package deck

import "math/big"

const (
	legendSize   = 9
	legendHeight = 16
	swatch       = 10
	scaleTop     = 20
	plotTop      = 38
	labelHeight  = 16
	labelSize    = 8
	nothingDrawn = "No amounts to draw"
)

func (s *sheet) chart(c Chart, at area) {
	s.legend(c.Series, at)
	s.texts = append(s.texts, text{X: at.x, Y: at.y + scaleTop, W: at.w, H: noteHeight, Size: labelSize, Colour: Subtle,
		Value: fit(c.MaxLabel, at.w, labelSize, false)})
	plot := area{x: at.x, y: at.y + plotTop, w: at.w, h: at.h - plotTop - labelHeight}
	s.boxes = append(s.boxes,
		box{X: plot.x, Y: plot.y, W: plot.w, H: 1, Fill: Hairline},
		box{X: plot.x, Y: plot.y + plot.h, W: plot.w, H: 1, Fill: Ink})
	if len(c.Categories) == 0 || len(c.Series) == 0 {
		return
	}
	top := largest(c.Series)
	if top <= 0 {
		s.texts = append(s.texts, text{X: plot.x, Y: plot.y + plot.h/2, W: plot.w, H: noteHeight, Size: noteSize,
			Colour: Subtle, Align: Center, Value: nothingDrawn})
	}
	group := plot.w / len(c.Categories)
	for i, category := range c.Categories {
		x := plot.x + i*group
		if top > 0 {
			s.bars(c.Series, i, top, area{x: x, y: plot.y, w: group, h: plot.h})
		}
		s.texts = append(s.texts, text{X: x, Y: plot.y + plot.h + 3, W: group, H: labelHeight - 3, Size: labelSize,
			Colour: Ink, Align: Center, Value: fit(category, group-2, labelSize, false)})
	}
}

func (s *sheet) legend(series []Series, at area) {
	x := at.x
	for _, one := range series {
		width := textWidth(one.Name, legendSize, false)
		s.boxes = append(s.boxes, box{X: x, Y: at.y + 3, W: swatch, H: swatch, Fill: one.Colour})
		s.texts = append(s.texts, text{X: x + swatch + 4, Y: at.y, W: width + 2, H: legendHeight, Size: legendSize,
			Colour: Ink, Value: one.Name})
		x += swatch + 4 + width + sectionGap
	}
}

// bars draws the bars of category i in its group: a quarter of the group is space, the rest is shared by the series.
func (s *sheet) bars(series []Series, i int, top int64, group area) {
	gap := max(group.w/4, 2)
	width := max((group.w-gap)/len(series), 1)
	for j, one := range series {
		if i >= len(one.Values) || one.Values[i] <= 0 {
			continue
		}
		height := scaled(one.Values[i], top, group.h)
		s.boxes = append(s.boxes, box{X: group.x + gap/2 + j*width, Y: group.y + group.h - height, W: width, H: height,
			Fill: one.Colour})
	}
}

func largest(series []Series) int64 {
	var top int64
	for _, one := range series {
		for _, v := range one.Values {
			top = max(top, v)
		}
	}
	return top
}

// scaled is value ÷ top × length, in whole points; big integers keep any amount of cents from overflowing.
func scaled(value, top int64, length int) int {
	product := new(big.Int).Mul(big.NewInt(value), big.NewInt(int64(length)))
	return int(product.Quo(product, big.NewInt(top)).Int64())
}
