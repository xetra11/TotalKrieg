// Package campaign stores TotalKrieg campaigns as JSON files, so a player can
// leave a match and continue it later.
package campaign

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xetra11/TotalKrieg/internal/game"
)

// Version is the file layout of a campaign save.
const Version = 1

// Record is one saved campaign: the file's metadata and the match itself.
type Record struct {
	ID      string
	Name    string
	Created time.Time
	Updated time.Time
	State   game.Snapshot
}

// Summary is what the menu needs to list a campaign without loading the match.
type Summary struct {
	ID      string
	Name    string
	Faction game.Faction
	Round   int
	Over    bool
	Winner  game.Faction
	Reason  string
	Updated time.Time
}

// Summary describes the record for the menu.
func (r Record) Summary() Summary {
	return Summary{
		ID:      r.ID,
		Name:    r.Name,
		Faction: r.State.First,
		Round:   r.State.Round,
		Over:    r.State.Over,
		Winner:  r.State.Winner,
		Reason:  r.State.Reason,
		Updated: r.Updated,
	}
}

// Store keeps campaigns in one folder.
type Store struct {
	Dir string
}

// DefaultDir is the user's campaign folder, next to the other TotalKrieg data.
func DefaultDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "TotalKrieg", "campaigns"), nil
}

// NewStore returns a store for a folder. The folder is created on the first
// save, not now.
func NewStore(dir string) *Store {
	return &Store{Dir: dir}
}

// New creates a campaign for a fresh match and writes it. The name becomes the
// file name, sanitized for the filesystem.
func (s *Store) New(name string, st *game.State) (Record, error) {
	snap, err := st.Snapshot()
	if err != nil {
		return Record{}, err
	}
	id := sanitizeName(name)
	if id == "" {
		id = "campaign"
	}
	// Make unique if a campaign with this name exists
	if _, err := s.Load(id); err == nil {
		id = s.uniqueID(id)
	}
	now := time.Now().UTC()
	rec := Record{
		ID:      id,
		Name:    name,
		Created: now,
		Updated: now,
		State:   snap,
	}
	return rec, s.Save(&rec)
}

// Save writes a campaign, replacing an older file with the same ID.
func (s *Store) Save(rec *Record) error {
	if rec == nil {
		return errors.New("no campaign to save")
	}
	if rec.State.Version == 0 {
		return errors.New("campaign has no match")
	}
	if _, err := s.path(rec.ID); err != nil {
		return err
	}
	if rec.Created.IsZero() {
		rec.Created = time.Now().UTC()
	}
	rec.Updated = time.Now().UTC()

	data, err := json.MarshalIndent(fileOf(*rec), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	path, _ := s.path(rec.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Load reads one campaign.
func (s *Store) Load(id string) (Record, error) {
	path, err := s.path(id)
	if err != nil {
		return Record{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return Record{}, fmt.Errorf("campaign %s: %w", id, err)
	}
	if f.Version != Version {
		return Record{}, fmt.Errorf("campaign %s has unsupported version %d", id, f.Version)
	}
	if f.ID == "" {
		f.ID = id
	}
	if f.ID != id {
		return Record{}, fmt.Errorf("campaign %s contains %q", id, f.ID)
	}
	return f.record(), nil
}

// List returns the campaigns, most recently saved first. Files that can't be
// read are skipped, so one broken save doesn't hide the others.
func (s *Store) List() ([]Summary, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Summary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		rec, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, rec.Summary())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Updated.After(out[j].Updated)
	})
	return out, nil
}

// Delete removes a campaign.
func (s *Store) Delete(id string) error {
	path, err := s.path(id)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// path keeps an ID from reaching outside the campaign folder.
func (s *Store) path(id string) (string, error) {
	if id == "" || id == "." || id == ".." || filepath.IsAbs(id) || id != filepath.Base(id) {
		return "", fmt.Errorf("invalid campaign id %q", id)
	}
	return filepath.Join(s.Dir, id+".json"), nil
}

func (s *Store) uniqueID(base string) string {
	id := base
	for n := 2; ; n++ {
		if _, err := os.Stat(filepath.Join(s.Dir, id+".json")); err != nil {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

// sanitizeName turns a campaign name into a safe file name.
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	// Replace spaces and special characters with dash
	var out strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
		} else if out.Len() > 0 && out.String()[out.Len()-1] != '-' {
			out.WriteRune('-')
		}
	}
	result := strings.Trim(out.String(), "-")
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

// file is the JSON layout of a campaign.
type file struct {
	Version int           `json:"version"`
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Created time.Time     `json:"created"`
	Updated time.Time     `json:"updated"`
	State   game.Snapshot `json:"state"`
}

func fileOf(rec Record) file {
	return file{
		Version: Version,
		ID:      rec.ID,
		Name:    rec.Name,
		Created: rec.Created,
		Updated: rec.Updated,
		State:   rec.State,
	}
}

func (f file) record() Record {
	return Record{
		ID:      f.ID,
		Name:    f.Name,
		Created: f.Created,
		Updated: f.Updated,
		State:   f.State,
	}
}
