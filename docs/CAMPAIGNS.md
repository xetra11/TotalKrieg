# Campaign System

TotalKrieg now has a complete campaign save system that allows players to:

## Features

### 1. Named Campaigns
- Enter a custom name for each campaign
- Names are used as file names (sanitized for filesystem safety)
- Multiple campaigns with the same name get automatic suffixes (-2, -3, etc.)

### 2. Campaign Management
- **Main Menu**: Shows list of all saved campaigns
- **Continue**: Click any campaign to resume where you left off
- **New Campaign**: Start a fresh campaign with a custom name

### 3. Automatic Saving
The game automatically saves after:
- Every completed turn (both player and AI)
- Every battle resolution (manual or automatic)
- Closing the game window

### 4. Campaign Information
Each saved campaign shows:
- Campaign name
- Faction (Wehrmacht or Soviet Union)
- Current round number
- Last saved timestamp
- Status (in progress or finished)

### 5. Complete State Preservation
The save system preserves:
- All units and their positions
- Order of battle (formations and companies)
- Planned orders and pending retreats
- Resources and manpower
- Combat reports
- Random number generator state (for deterministic gameplay)

## Storage

Campaign files are stored as JSON in:
- **Windows**: `%APPDATA%\TotalKrieg\campaigns\`
- **Linux/Mac**: `~/.config/TotalKrieg/campaigns/`

Each campaign is a single file named after the campaign (e.g., `wehrmacht-1941.json`).

## Menu Navigation

### Starting a New Campaign
1. Click "New campaign" in the main menu
2. Enter a campaign name
3. Click "Continue"
4. Choose your faction (Wehrmacht or Soviet Union)
5. Game starts and is immediately saved

### Continuing a Campaign
1. Click on any campaign in the main menu list
2. Game resumes at the exact state it was saved

### Controls
- **ESC**: Navigate back through menus
- **Enter**: Confirm in dialogs
- Campaign name supports letters, numbers, spaces, dashes, and underscores

## Technical Details

- **File Format**: JSON with versioning for forward compatibility
- **Atomicity**: Saves use temporary files and atomic rename to prevent corruption
- **Error Handling**: Corrupted saves are skipped, other campaigns remain accessible
- **Memory Efficient**: Only active campaign is loaded in memory

## Development

The campaign system consists of three main components:

1. **`internal/campaign`**: Campaign store and file management
2. **`internal/game/save.go`**: Game state serialization (Snapshot/Restore)
3. **`internal/ui`**: Menu system and autosave integration

All components have comprehensive test coverage.
