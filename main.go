// Carnival Constructor Classic (c) 2026 Ironite.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

func windowTitle() string { return productName + " (Demo) " + version }

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "/version") {
		fmt.Println(version)
		return
	}
	if runtime.GOOS != "js" {
		if logf, err := os.OpenFile(filepath.Join(os.TempDir(), "IroniteCarnivalConstructor.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
			log.SetOutput(logf)
		}
	}
	log.Printf("%s %s starting", productName, version)
	if !singleInstance() {
		return
	}
	loadAssets()
	g := newGame()
	if !strings.Contains(testQuery(), "mute") {
		snd.init()
	}
	snd.setVolume(float64(g.save.Settings.Vol) / 100)
	snd.setSfx(g.save.Settings.Sfx)
	snd.setMusic(g.save.Settings.Music)
	g.applyTestQuery(testQuery())

	ebiten.SetWindowTitle(windowTitle())
	if len(art.icons) > 0 {
		ebiten.SetWindowIcon(art.icons)
	}
	ebiten.SetWindowSize(1440, 900)
	ebiten.SetWindowSizeLimits(1024, 640, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(true)
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		fatal(productName + " could not start: " + err.Error())
	}
}
