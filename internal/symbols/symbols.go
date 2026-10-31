// Package symbols loads the unit symbol sets. Each faction has its own JSON
// file, so a corrected symbol needs no code change.
package symbols

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed wehrmacht.json sovietunion.json
var files embed.FS

// Shape is one drawing step in symbol coordinates.
type Shape struct {
	Op     string       `json:"op"` // line, outline or fill
	Points [][2]float32 `json:"points"`
	Width  float32      `json:"width"` // stroke width, 2 when unset
}

// Label is where the unit designation sits inside the symbol.
type Label struct {
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	Size float32 `json:"size"`
}

// Symbol is a drawn unit sign.
type Symbol struct {
	W      float32 `json:"w"`
	H      float32 `json:"h"`
	Shapes []Shape `json:"shapes"`
	Label  *Label  `json:"label"`
}

// Set maps unit kinds to their symbols.
type Set struct {
	Source string            `json:"source"`
	Units  map[string]Symbol `json:"units"`
}

// Load reads the set stored as <name>.json.
func Load(name string) (*Set, error) {
	data, err := files.ReadFile(name + ".json")
	if err != nil {
		return nil, err
	}
	set := &Set{}
	if err := json.Unmarshal(data, set); err != nil {
		return nil, fmt.Errorf("symbols %s: %w", name, err)
	}
	return set, nil
}
