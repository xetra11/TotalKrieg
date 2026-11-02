# Animation System

TotalKrieg now features smooth, animated unit movements that make the game more readable and enjoyable.

## Features

### Smooth Movement Animations
- **Duration**: Each move takes 20 frames (~0.33 seconds at 60 FPS)
- **Easing**: Cubic ease-out for natural deceleration
- **Interpolation**: Units smoothly glide between hexes

### Sequential Animations
- Moves are queued and played one after another
- **Delay**: 5 frames between consecutive moves
- Player can follow what's happening step by step

### Both Player and AI
- **Player Orders**: Animated when committing turn
- **AI Moves**: Each AI movement is shown individually
  - Defense movements
  - Attack movements
  - Advance movements
  - Staff repositioning

### Visual Clarity
- Units glide smoothly across the hex grid
- Current animation shows unit in transition
- Map updates are visible during animation
- No sudden teleportation

## Implementation

### Animation Queue
```
Move 1 → [20 frames animation] → [5 frames delay] →
Move 2 → [20 frames animation] → [5 frames delay] →
Move 3 → ...
```

### Turn Flow
1. Player commits orders
2. Player moves are recorded
3. Moves execute in game logic
4. Animations play sequentially
5. AI turn starts
6. AI moves recorded during execution
7. Animations play after each movement phase
8. Fights pause for player decisions

### Game Logic
- Game state updates immediately
- Animations are visual-only
- Unit positions reflect final state
- Animations don't block logic, only presentation

## Technical Details

### Components

**`animation.go`**
- Animation queue management
- Interpolation calculations
- Easing functions
- Progress tracking

**`moves.go`**
- Movement recording
- Animation triggering
- Queue management

**Modified Files**
- `state.go`: Records moves during execution
- `ai.go`: Yields moves for animation between phases
- `draw.go`: Uses interpolated positions for animating units
- `app.go`: Waits for animations between turns

### Performance
- Animations run at 60 FPS
- No impact on game logic performance
- Can be disabled for tests via `Options.NoAnimation`
- Minimal memory footprint

### Testing
- All animations can be disabled via `NoAnimation` flag
- Tests run at full speed without waiting
- Animation logic tested separately

## User Experience

### Benefits
1. **Clarity**: See exactly what moved where
2. **Pacing**: Game doesn't feel rushed
3. **Immersion**: Smoother, more polished feel
4. **Learning**: Easier to understand AI decisions

### What You'll Notice
- Units glide smoothly between hexes
- AI moves happen one at a time
- Clear visual feedback for all movements
- No confusing instant jumps
- Professional game feel

## Configuration

Animations are always enabled in normal play. For development and testing:

```go
opts := ui.Options{
    NoAnimation: true,  // Skip animations
}
```

This is automatically set in all tests for fast execution.
