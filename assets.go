package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed assets
var assetFS embed.FS

type assetSet struct {
	rye, oswald, rubik *text.GoTextFaceSource
	icons              []image.Image
	logo               *ebiten.Image
	white, glow        *ebiten.Image
}

var art assetSet

func mustFont(name string) *text.GoTextFaceSource {
	b, _ := assetFS.ReadFile("assets/fonts/" + name)
	s, err := text.NewGoTextFaceSource(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return s
}

func loadAssets() {
	art.rye = mustFont("Rye.ttf")
	art.oswald = mustFont("Oswald.ttf")
	art.rubik = mustFont("Rubik.ttf")
	for _, n := range []string{"16", "32", "48", "64", "256"} {
		if b, err := assetFS.ReadFile("assets/icon-" + n + ".png"); err == nil {
			if im, _, err := image.Decode(bytes.NewReader(b)); err == nil {
				art.icons = append(art.icons, im)
			}
		}
	}
	if b, err := assetFS.ReadFile("assets/ironite-logo.png"); err == nil {
		if im, _, err := image.Decode(bytes.NewReader(b)); err == nil {
			art.logo = ebiten.NewImageFromImage(im)
		}
	}
	w := ebiten.NewImage(3, 3)
	w.Fill(color.White)
	art.white = w.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	art.glow = makeGlow()
}
