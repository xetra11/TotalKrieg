# TotalKrieg

Turn-based operational game on a hex map: Wehrmacht against the Soviet Union, player against AI.
Prototype in Go with [Ebitengine](https://ebitengine.org). The rules are in [docs/GDD.md](docs/GDD.md).

## Installing

Download `TotalKrieg-setup.exe` from the repository's [releases](https://github.com/xetra11/TotalKrieg/releases). It installs for the current user, without admin rights, and brings along MiusMissionMaker and Node.js. Only Mius-Front itself has to be installed.

Every pushed tag `v*` runs the [Installer workflow](.github/workflows/installer.yml), which builds the installer with the latest MiusMissionMaker release and publishes it as a release of that tag:

```bash
git tag v1.2.3
```

```bash
git push origin v1.2.3
```

To build the installer locally, with Go, Node.js and Inno Setup 6:

```bash
powershell -File installer/build.ps1 -Version 1.2.3
```

The result is `dist/TotalKrieg-setup.exe`.

## Running

Requires Go 1.26 or newer.

```bash
go run ./cmd/totalkrieg
```

Options:

| Option | Effect |
| --- | --- |
| `-seed 7` | Plays a specific map, 0 picks at random |
| `-faction wehrmacht` | Skips the menu, also `sovietunion` |
| `-autoplay 18` | Plays this many AI turns before the start |
| `-shot image.png` | Writes an image and quits |
| `-missionmaker folder` | MiusMissionMaker folder, empty looks next to the program, then in `../MiusMissionMaker` |
| `-miusgame folder` | Mius-Front folder, empty uses the Steam default |

## Battles in Mius-Front

Battles in Mius-Front need MiusMissionMaker, Node.js and the installed game. The game must be closed before the start. The battle dialog first shows the battle line-up, and you start Mius-Front from there. When the battle ends, TotalKrieg reads the outcome and losses by itself, shows them per platoon and books them with "Apply result". If that fails, you enter the outcome by hand. Generated scenarios and missions land in the `missions/` folder, for an installed game in `%LOCALAPPDATA%\TotalKrieg\missions`.

TotalKrieg needs the Mius-Front folder to start a battle and to read its result. It expects the default Steam path `C:/Program Files (x86)/Steam/steamapps/common/Graviteam Tactics Mius-Front`. If the game is somewhere else, for example in another Steam library, pass the folder that contains `starter_x64.exe` with `-miusgame`. For an installed game, add it to the end of **Target** in the shortcut's properties, after the closing quote:

```text
"%LOCALAPPDATA%\Programs\TotalKrieg\TotalKrieg.exe" -miusgame "D:\SteamLibrary\steamapps\common\Graviteam Tactics Mius-Front"
```

## Controls

During your turn you act in any order. Income comes at the start of a turn, captures count at the end of a turn.

- **Raise:** Click one of your cities, then choose a unit on the right.
- **Move:** Click one of your units, then a marked hex. Leftover movement stays for further steps.
- **Attack:** Click one of your companies, then an enemy outlined in red. A dialog asks how to resolve the battle.
- **Flee:** When your unit is beaten, click one of the marked hexes for it to flee to.
- **Order of battle:** Click a unit, then **G** or the button on the right. The window shows the battalion with callsigns, company leaders, positions and the counterpart in Mius-Front.
- **Staffs:** Companies move 3 hexes only within reach of their staff, otherwise 1 hex. A red dot marks a missing link. Staffs don't attack or capture.
- **Enter** ends the turn or confirms the automatic resolution, **Esc** cancels or clears the selection.

## Structure

| Path | Content |
| --- | --- |
| `cmd/totalkrieg` | Program start |
| `internal/game` | Rules, map generation and AI, without graphics |
| `internal/mission` | Adapter to MiusMissionMaker |
| `internal/symbols` | Symbol sets per faction as JSON |
| `internal/ui` | User interface with Ebitengine |
| `concept` | Concept art and its source, in German |
| `docs` | GDD and sources |
| `installer` | Installer script and build |
| `tools` | Concept art export |

## Tests

```bash
go test ./...
```

With `TK_MISSIONMAKER` set, `TestGeneratorAcceptsScenarios` builds all battle sizes with the real generator, without starting the game.

`TestFullMatches` lets the AI play 400 matches against itself. With `go test ./internal/game -run TestFullMatches -v` the test shows the distribution of wins.
