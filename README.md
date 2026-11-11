# Carnival Constructor Classic (Demo)

A traveling carnival sim by Ironite, written in Go with [Ebitengine](https://ebitengine.org).
Build rides, midway games and food stands on a fairground lot, open the gates at 6 PM, and keep the
crowd happy (and spending) until 11. This is the free demo: the Pine Hollow County Fair, night after night, for as long as you can keep it going.
It ships on its own and inside the [Ironite Launcher](https://github.com/adamglassner/launcher), where it's
listed under Simulation as a DEMO.

## How to play

- **Daytime:** pick something from the bar at the bottom and click the lot to build it. Guests line up at
  the gold square in front. Click anything you've built to set its price, set a midway game's prize odds,
  or sell it.
- **Paths:** on the Paths tab, click and drag to lay footpaths. Guests keep to them, walk faster and stay happier.
- **Night:** press **Open the Gates**. Guests pay admission, then ride, play, eat, rest and drop litter.
  Space pauses; 1, 2 and 3 set the speed.
- **Fair or rigged:** rigging a game pays out fewer prizes, but guests catch on and your reputation drops.
  Reputation decides how many guests come the next night.
- Rides break down now and then: click the ride and send a mechanic.
- Esc cancels or opens the menu, M toggles music, F11 toggles full screen, Delete sells what's selected.

Settings, records and achievements are kept in `%APPDATA%\Ironite\CarnivalConstructor\save.json`, the run in
progress (saved every morning, for Continue) in `run.json` beside it, and the
achievements Ironite Launcher shows in `%APPDATA%\Ironite\Achievements\carnival.json`.

## Play

The easiest way is **[Ironite Launcher](https://github.com/adamglassner/launcher)**: it downloads the game from
this repository's releases, installs it and keeps it up to date.

Or download from the [latest release](../../releases/latest): `CarnivalConstructor-Setup-v<version>.exe`
installs it with Start menu and desktop shortcuts, or unzip `CarnivalConstructor-<version>-windows.zip`
anywhere and run `CarnivalConstructor.exe`.

## Code

| File | What it is |
|---|---|
| `internal/carnival/` | The simulation: the lot, attractions, guests, money, reputation and nights (no graphics; has tests). |
| `internal/synth/` | Synthesizes the calliope waltz and the sound effects at start-up (has tests). |
| `main.go`, `game.go` | Window setup, input, the game loop and screenshots hooks. |
| `scene.go` | Draws the lot, the rides, games, food trucks and guests, and the night lighting. |
| `ui.go`, `screens.go` | The top bar, side panel, build bar; title, menu, help, settings and end-of-night screens. |
| `draw.go` | Vector drawing and text on a supersampled canvas. |
| `sound.go`, `save.go` | Audio playback; the save file and achievements. |
| `tools/art/` | The icon (SVG) and the script that renders it for the game, the .exe and the installer. |
| `tools/mkres/` | Writes the Windows icon and version info into the .exe. |

## Building

```bash
bash build.sh          # needs Go 1.24+; output in dist/CarnivalConstructor/
go test ./internal/... # simulation and audio tests
```

## Releasing

Add a new section at the top of `CHANGELOG.md` with the new version and push to `main`. GitHub Actions tests,
builds and publishes `v<version>` with the zip the launcher installs and a standalone setup.

© 2026 Ironite. All rights reserved. Typefaces: Rye, Oswald and Rubik (SIL Open Font License, see
`assets/fonts/OFL.txt`).
