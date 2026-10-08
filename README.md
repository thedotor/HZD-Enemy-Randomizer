# HZD Enemy Randomizer (Map)

A map-based enemy randomizer for **Horizon Zero Dawn Complete Edition** on PC (the original
release, not Remastered). Pick areas on the game's own world map and choose which machines and
human enemies appear there.

!\[icon](tools/winres/icon\_1024.png)

## Features

* Randomize machines by area, by ready-made region, or across the whole map, with seeds,
swap chance and difficulty (Grouped = similar size, Chaotic = anything)
* Roaming herds and patrols, herd size, Frozen Wilds machines anywhere
* Optional: Cauldrons, story missions, Hunting Grounds, bosses, Rockbreaker / Corruptor /
Deathbringer anywhere
* Human enemies (bandits, Eclipse, Frozen Wilds bandits), and humans ⇄ machines swaps
* Health and damage sliders per size group, per machine and per human faction/type
* Pick the machine for any spot by clicking it on the map
* Preview of what a setup will put in the world, presets with shareable codes, recent builds

Full user guide: [docs/README\_Map.txt](docs/README_Map.txt) ·
History: [docs/CHANGELOG\_Map.txt](docs/CHANGELOG_Map.txt)

## Download

Get `HZD\_EnemyRandomizer\_Map.exe` from the [Releases](../../releases) page.
Requires Horizon Zero Dawn Complete Edition (original PC version, latest update) on Windows.

## No game files included

This repository and the program contain **no files from the game**. On first start the program
reads what it needs (map tiles, map icons, spawn and level data) from the player's own
installation, using the game's own `oo2core\_\*\_win64.dll` to unpack the archives, and checks each
file against the SHA-256 listed in [`mapdata/manifest.json`](mapdata/manifest.json).
`mapdata/sites.json` holds only positions, file names and offsets of spawn spots.

## Building

See [BUILDING.txt](BUILDING.txt). In short, with Go 1.24.7:

```
GOOS=windows GOARCH=amd64 go build -trimpath -o HZD\_EnemyRandomizer\_Map.exe .
```

The build is reproducible: the same Go version gives a byte-identical exe. Releases are built by
the public [GitHub Actions workflow](.github/workflows/build.yml).

## Code signing policy

Free code signing provided by [SignPath.io](https://about.signpath.io/), certificate by
[SignPath Foundation](https://signpath.org/). Details: [CODE\_SIGNING\_POLICY.md](CODE_SIGNING_POLICY.md).

* Committers and reviewers: thedotor
* Approvers: [thedotor](https://github.com/YOUR-GITHUB-USERNAME)

## Privacy

This program will not transfer any information to other networked systems unless specifically
requested by the user or the person installing or operating it. See [PRIVACY.md](PRIVACY.md).

## AI disclosure

Most of this program's code was written with an AI assistant (Claude), directed and tested
in-game by the project owner.

## Licence and disclaimer

[MIT](LICENSE). Free fan-made tool, not affiliated with or endorsed by Guerrilla Games or Sony
Interactive Entertainment. Horizon Zero Dawn is a trademark of its owners.

