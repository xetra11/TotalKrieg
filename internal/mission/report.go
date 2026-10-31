package mission

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xetra11/TotalKrieg/internal/game"
)

const (
	defaultGameDir = "C:/Program Files (x86)/Steam/steamapps/common/Graviteam Tactics Mius-Front"
	// resultFile is what the battle leaves for the operational game. The game
	// deletes it when a battle loads and writes it when the battle ends.
	resultFile = "data/common/shared/config/tacback_image.cfgpack"
	engineLog  = "out/common_engine.log"

	// platoonMask cuts a squad id down to the id of its platoon.
	platoonMask = ^uint32(0x1ff)
	// drawCode is the result between defeats and victories. Seen so far are
	// 2 for a minor defeat and 4 for a minor victory.
	drawCode = 3
)

var resultLine = regexp.MustCompile(`Tactical result : (.+) \[(\d+)\]`)

// ReadResult returns the outcome of the battle started at since, or nil while
// it is still running.
func ReadResult(c Config, s Scenario, since time.Time) (*game.FieldResult, error) {
	dir := c.GameDir
	if dir == "" {
		dir = defaultGameDir
	}
	path := filepath.Join(dir, resultFile)
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.ModTime().Before(since) {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	log, err := os.ReadFile(filepath.Join(dir, engineLog))
	if err != nil {
		return nil, err
	}
	return parseResult(data, string(log), s.Roster)
}

// parseResult joins the result file and the log. The file lists squads platoon
// by platoon in mission order, so the n-th platoon is the n-th roster entry.
func parseResult(data []byte, log string, roster []RosterEntry) (*game.FieldResult, error) {
	text, code, err := tacticalResult(log)
	if err != nil {
		return nil, err
	}
	tables, err := readTables(data)
	if err != nil {
		return nil, err
	}
	squads, lost := tables["squads"], map[uint32]bool{}
	if squads == nil {
		return nil, errors.New("result file without squads")
	}
	if rows := tables["lost_squads"]; rows != nil {
		for _, row := range rows {
			lost[row[0].u32(0)] = true
		}
	}

	var order []uint32
	sum := map[uint32]float64{}
	count, gone := map[uint32]int{}, map[uint32]int{}
	for _, row := range squads {
		id := row[0].u32(0)
		platoon := id & platoonMask
		if count[platoon] == 0 {
			order = append(order, platoon)
		}
		count[platoon]++
		if lost[id] {
			gone[platoon]++
		} else {
			sum[platoon] += float64(row[1].f32(8))
		}
	}
	if len(order) != len(roster) {
		return nil, fmt.Errorf("result with %d platoons, the mission had %d", len(order), len(roster))
	}

	r := &game.FieldResult{Text: text, Outcome: outcomeOf(code), Remaining: map[*game.Element]float64{}}
	total, platoons := map[*game.Element]float64{}, map[*game.Element]int{}
	for i, id := range order {
		owner := roster[i].Owner
		total[owner] += sum[id] / float64(count[id])
		platoons[owner]++
		p := game.PlatoonLoss{Owner: owner, Name: roster[i].Label, Squads: count[id], Lost: gone[id]}
		if alive := count[id] - gone[id]; alive > 0 {
			p.Condition = sum[id] / float64(alive)
		}
		r.Platoons = append(r.Platoons, p)
	}
	for e, v := range total {
		r.Remaining[e] = v / float64(platoons[e])
	}
	return r, nil
}

func tacticalResult(log string) (string, int, error) {
	matches := resultLine.FindAllStringSubmatch(log, -1)
	if len(matches) == 0 {
		return "", 0, errors.New("no result in the log")
	}
	last := matches[len(matches)-1]
	code, err := strconv.Atoi(last[2])
	return strings.TrimSpace(last[1]), code, err
}

func outcomeOf(code int) game.Outcome {
	switch {
	case code > drawCode:
		return game.AttackerWins
	case code < drawCode:
		return game.DefenderWins
	}
	return game.Draw
}

// cell is one table value of an AZCF config: 16 raw bytes.
type cell []byte

func (c cell) u32(off int) uint32 {
	return binary.LittleEndian.Uint32(c[off:])
}

func (c cell) f32(off int) float32 {
	return math.Float32frombits(c.u32(off))
}

const (
	azcfHeader   = 0x20
	azcfNodeSize = 0x80
	azcfCellSize = 0x20
	azcfTable    = 2
)

// readTables reads the table nodes of an AZCF config. The format is described
// in tools/azcf.js of MiusMissionMaker.
func readTables(b []byte) (map[string][][]cell, error) {
	if len(b) < azcfHeader || string(b[:4]) != "AZCF" {
		return nil, errors.New("not an AZCF file")
	}
	nodes := int(binary.LittleEndian.Uint32(b[4:]))
	if azcfHeader+nodes*azcfNodeSize > len(b) {
		return nil, errors.New("AZCF file is truncated")
	}
	out := map[string][][]cell{}
	for i := 0; i < nodes; i++ {
		rec := b[azcfHeader+i*azcfNodeSize:]
		field := func(n int) int { return int(binary.LittleEndian.Uint32(rec[0x40+n*4:])) }
		if field(0) != azcfTable {
			continue
		}
		cols, rows, data := field(1), field(2), field(4)
		if data+cols*rows*azcfCellSize > len(b) {
			return nil, errors.New("AZCF table is truncated")
		}
		table := make([][]cell, rows)
		for r := range table {
			table[r] = make([]cell, cols)
			for c := range table[r] {
				start := data + (r*cols+c)*azcfCellSize
				table[r][c] = cell(b[start : start+16])
			}
		}
		out[nodeName(rec)] = table
	}
	return out, nil
}

func nodeName(rec []byte) string {
	name := rec[0x20:0x40]
	if end := strings.IndexByte(string(name), 0); end >= 0 {
		name = name[:end]
	}
	return string(name)
}
