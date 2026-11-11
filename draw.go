package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Everything is laid out on a 1440x900 logical canvas and drawn onto an
// offscreen canvas at up to twice the screen's resolution, which is then
// scaled down: one cheap pass of anti-aliasing for every shape and letter.
const logicalW, logicalH = 1440, 900

type fontKind int

const (
	fDisplay fontKind = iota // Rye (titles)
	fHead                    // Oswald
	fBody                    // Rubik
)

type font struct {
	k    fontKind
	size float64
	wght float64
}

func display(size float64) font    { return font{fDisplay, size, 400} }
func head(size float64) font       { return font{fHead, size, 600} }
func body(size, wght float64) font { return font{fBody, size, wght} }

var wghtTag = text.MustParseTag("wght")

type faceKey struct {
	k        fontKind
	px, wght float64
}

var faceCache = map[faceKey]*text.GoTextFace{}

func face(f font, s float64) *text.GoTextFace {
	key := faceKey{f.k, math.Round(f.size*s*4) / 4, f.wght}
	if fc, ok := faceCache[key]; ok {
		return fc
	}
	src := art.rubik
	switch f.k {
	case fDisplay:
		src = art.rye
	case fHead:
		src = art.oswald
	}
	fc := &text.GoTextFace{Source: src, Size: math.Max(1, key.px)}
	if f.k != fDisplay {
		fc.SetVariation(wghtTag, float32(f.wght))
	}
	faceCache[key] = fc
	return fc
}

// pen draws onto dst in logical coordinates.
type pen struct {
	dst    *ebiten.Image
	s      float64
	ox, oy float64
	alpha  float64
}

func (p pen) X(x float64) float32 { return float32((x + p.ox) * p.s) }
func (p pen) Y(y float64) float32 { return float32((y + p.oy) * p.s) }
func (p pen) L(v float64) float32 { return float32(v * p.s) }
func (p pen) fade(a float64) pen  { p.alpha *= a; return p }

func (p pen) c(c color.Color) color.Color {
	if p.alpha >= 1 {
		return c
	}
	r, g, b, a := c.RGBA()
	k := p.alpha
	return color.RGBA64{uint16(float64(r) * k), uint16(float64(g) * k), uint16(float64(b) * k), uint16(float64(a) * k)}
}

func (p pen) rect(x, y, w, h float64, c color.Color) {
	vector.DrawFilledRect(p.dst, p.X(x), p.Y(y), p.L(w), p.L(h), p.c(c), false)
}

func (p pen) strokeRect(x, y, w, h, lw float64, c color.Color) {
	vector.StrokeRect(p.dst, p.X(x), p.Y(y), p.L(w), p.L(h), p.L(lw), p.c(c), false)
}

func (p pen) circle(x, y, r float64, c color.Color) {
	vector.DrawFilledCircle(p.dst, p.X(x), p.Y(y), p.L(r), p.c(c), false)
}

func (p pen) ring(x, y, r, lw float64, c color.Color) {
	vector.StrokeCircle(p.dst, p.X(x), p.Y(y), p.L(r), p.L(lw), p.c(c), false)
}

func (p pen) line(x1, y1, x2, y2, lw float64, c color.Color) {
	vector.StrokeLine(p.dst, p.X(x1), p.Y(y1), p.X(x2), p.Y(y2), p.L(lw), p.c(c), false)
}

func (p pen) fill(path *vector.Path, c color.Color) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	r, g, b, a := p.c(c).RGBA()
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		if a > 0 {
			vs[i].ColorR = float32(r) / float32(a)
			vs[i].ColorG = float32(g) / float32(a)
			vs[i].ColorB = float32(b) / float32(a)
		}
		vs[i].ColorA = float32(a) / 0xffff
	}
	p.dst.DrawTriangles(vs, is, art.white, &ebiten.DrawTrianglesOptions{FillRule: ebiten.FillRuleNonZero})
}

func (p pen) stroke(path *vector.Path, lw float64, c color.Color) {
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: p.L(lw), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound})
	r, g, b, a := p.c(c).RGBA()
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		if a > 0 {
			vs[i].ColorR = float32(r) / float32(a)
			vs[i].ColorG = float32(g) / float32(a)
			vs[i].ColorB = float32(b) / float32(a)
		}
		vs[i].ColorA = float32(a) / 0xffff
	}
	p.dst.DrawTriangles(vs, is, art.white, &ebiten.DrawTrianglesOptions{})
}

// poly fills a polygon given as x, y pairs.
func (p pen) poly(c color.Color, pts ...float64) {
	var path vector.Path
	path.MoveTo(p.X(pts[0]), p.Y(pts[1]))
	for i := 2; i+1 < len(pts); i += 2 {
		path.LineTo(p.X(pts[i]), p.Y(pts[i+1]))
	}
	path.Close()
	p.fill(&path, c)
}

func (p pen) rrectPath(x, y, w, h, r float64) *vector.Path {
	r = math.Min(r, math.Min(w, h)/2)
	var path vector.Path
	path.MoveTo(p.X(x+r), p.Y(y))
	path.LineTo(p.X(x+w-r), p.Y(y))
	path.ArcTo(p.X(x+w), p.Y(y), p.X(x+w), p.Y(y+r), p.L(r))
	path.LineTo(p.X(x+w), p.Y(y+h-r))
	path.ArcTo(p.X(x+w), p.Y(y+h), p.X(x+w-r), p.Y(y+h), p.L(r))
	path.LineTo(p.X(x+r), p.Y(y+h))
	path.ArcTo(p.X(x), p.Y(y+h), p.X(x), p.Y(y+h-r), p.L(r))
	path.LineTo(p.X(x), p.Y(y+r))
	path.ArcTo(p.X(x), p.Y(y), p.X(x+r), p.Y(y), p.L(r))
	path.Close()
	return &path
}

func (p pen) rrect(x, y, w, h, r float64, c color.Color) { p.fill(p.rrectPath(x, y, w, h, r), c) }

func (p pen) rrectStroke(x, y, w, h, r, lw float64, c color.Color) {
	p.stroke(p.rrectPath(x, y, w, h, r), lw, c)
}

func (p pen) ellipse(cx, cy, rx, ry float64, c color.Color) {
	var path vector.Path
	const n = 28
	for i := 0; i <= n; i++ {
		a := float64(i) / n * 2 * math.Pi
		x, y := p.X(cx+math.Cos(a)*rx), p.Y(cy+math.Sin(a)*ry)
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	path.Close()
	p.fill(&path, c)
}

type align int

const (
	alLeft align = iota
	alCenter
	alRight
)

// text draws s with its baseline-ish middle at y (vertically centred on y).
func (p pen) text(s string, f font, x, y float64, al align, c color.Color) {
	fc := face(f, p.s)
	op := &text.DrawOptions{}
	op.LayoutOptions.PrimaryAlign = [...]text.Align{text.AlignStart, text.AlignCenter, text.AlignEnd}[al]
	op.LayoutOptions.SecondaryAlign = text.AlignCenter
	op.GeoM.Translate(float64(p.X(x)), float64(p.Y(y)))
	op.ColorScale.ScaleWithColor(p.c(c))
	op.Filter = ebiten.FilterLinear
	text.Draw(p.dst, s, fc, op)
}

func textW(s string, f font) float64 {
	fc := face(f, 1)
	w, _ := text.Measure(s, fc, 0)
	return w
}

// wrap splits s into lines no wider than w.
func wrap(s string, f font, w float64) []string {
	var lines []string
	for _, para := range splitLines(s) {
		words := splitWords(para)
		cur := ""
		for _, wd := range words {
			try := wd
			if cur != "" {
				try = cur + " " + wd
			}
			if cur != "" && textW(try, f) > w {
				lines = append(lines, cur)
				cur = wd
			} else {
				cur = try
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

func (p pen) para(s string, f font, x, y, w, lh float64, c color.Color) float64 {
	for _, l := range wrap(s, f, w) {
		p.text(l, f, x, y, alLeft, c)
		y += lh
	}
	return y
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}

func splitWords(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// glow draws a soft additive light at (x, y).
func (p pen) glow(x, y, r float64, c color.RGBA, k float64) {
	op := &ebiten.DrawImageOptions{}
	sz := float64(art.glow.Bounds().Dx())
	op.GeoM.Translate(-sz/2, -sz/2)
	op.GeoM.Scale(float64(p.L(r*2))/sz, float64(p.L(r*2))/sz)
	op.GeoM.Translate(float64(p.X(x)), float64(p.Y(y)))
	op.ColorScale.Scale(float32(c.R)/255*float32(k*p.alpha), float32(c.G)/255*float32(k*p.alpha), float32(c.B)/255*float32(k*p.alpha), float32(k*p.alpha))
	op.Blend = ebiten.BlendLighter
	op.Filter = ebiten.FilterLinear
	p.dst.DrawImage(art.glow, op)
}

func makeGlow() *ebiten.Image {
	const n = 128
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)-n/2+.5, float64(y)-n/2+.5) / (n / 2)
			v := math.Max(0, 1-d)
			v = v * v * (3 - 2*v)
			a := uint8(v * 255)
			img.Set(x, y, color.RGBA{a, a, a, a})
		}
	}
	return ebiten.NewImageFromImage(img)
}

func hex(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func withA(c color.RGBA, a float64) color.RGBA {
	k := math.Max(0, math.Min(1, a))
	return color.RGBA{uint8(float64(c.R) * k), uint8(float64(c.G) * k), uint8(float64(c.B) * k), uint8(255 * k)}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

// The palette from the concept art.
var (
	cNavy   = hex(0x14183a)
	cNight  = hex(0x0b0e27)
	cPanel  = hex(0x1c2150)
	cPanel2 = hex(0x262c63)
	cLine   = hex(0x3a3f78)
	cRed    = hex(0xe23b3b)
	cRedDk  = hex(0xb02424)
	cCream  = hex(0xfff4dc)
	cTan    = hex(0xe8d9b5)
	cMuted  = hex(0xc9bfa6)
	cGold   = hex(0xf4b942)
	cTeal   = hex(0x2bb3a3)
	cMint   = hex(0x5fd6c6)
	cPlum   = hex(0x5b2b4a)
	cPlumDk = hex(0x3a1b30)
	cGrass  = hex(0x3b7547)
	cGrass2 = hex(0x37703f)
	cDirt   = hex(0xc9a46e)
	cBrown  = hex(0x7a4b2a)
	cWhite  = color.RGBA{255, 255, 255, 255}
	cBlack  = color.RGBA{0, 0, 0, 255}
	cShadow = color.RGBA{0, 0, 0, 70}
	cPink   = hex(0xf7a8c4)
	cBlue   = hex(0x5e9ef0)
	cPurple = hex(0xc86ad8)
	cOrange = hex(0xe8873a)
	cBad    = hex(0xff8a7a)
)
