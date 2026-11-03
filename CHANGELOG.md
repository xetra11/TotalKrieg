# Changelog

All notable changes to TotalKrieg will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2024-11-03

### Added
- **Smooth unit movement animations** - Units now glide smoothly between hexes instead of teleporting
  - 20 frames per move (~0.33 seconds at 60 FPS)
  - Cubic ease-out interpolation for natural deceleration
  - Sequential animation with 5-frame delays between moves
  - Both player and AI moves are animated
- AI moves are now shown one at a time (defense, attack, advance, staff repositioning)
- Animation system with queue management (`internal/ui/animation.go`)
- Movement recording system (`internal/ui/moves.go`)
- Comprehensive animation documentation (`docs/ANIMATIONS.md`)
- Test mode with `NoAnimation` flag for fast test execution
- Unit move recording in game state for replay and animation
- **Version-numbered installer files** - Installers now use `totalkrieg_<version>.exe` format (e.g., `totalkrieg_0.2.0.exe`)

### Fixed
- Player movement animations now work correctly (were initially broken)
- Units no longer flash on their destination hex before animating
- Queued units stay at their start position until their animation plays
- Animation queue is now properly cleared when scripts end
- Tests updated to handle animation yields (skip nil fights)

### Technical
- `State.Move()` now returns the starting position for animation tracking
- `CommitTurn()` yields moves after company and staff phases for animation
- `AITurn()` yields moves after each movement phase
- `drawUnits()` uses interpolated positions for animating units
- New `positionOf()` method prevents visual glitches during queued animations

## [0.1.0] - 2024-11-03

### Added
- **Campaign save system** with named campaigns
  - User-entered campaign names
  - JSON-based save files in user config directory
  - Sanitized filenames from campaign names
  - Auto-save after each turn and battle resolution
- Campaign name input UI with text entry field
- Menu navigation: Main → Name Entry → Faction Selection
- Game state serialization with `Snapshot()`/`Restore()`
- RNG state serialization for deterministic replay
- Campaign management in `internal/campaign/campaign.go`

### Initial Features
- Hex-based tactical combat system
- Wehrmacht vs Soviet Union factions
- Company and Battalion units with organizational structure
- Movement and combat mechanics
- AI opponent with strategic priorities (defend, attack, advance)
- Cities, villages, factories as objectives
- Formation linking between companies and battalion staffs
- Resource management (manpower and materials)
- Turn-based gameplay with order planning phase

[0.2.0]: https://github.com/xetra11/TotalKrieg/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/xetra11/TotalKrieg/releases/tag/v0.1.0
