package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config locates MiusMissionMaker and the generated files.
type Config struct {
	ToolDir string // MiusMissionMaker project folder
	GameDir string // Mius-Front folder, empty for the Steam default
	OutDir  string // where scenarios and missions are written
}

// Available reports whether the MiusMissionMaker tools can be found.
func (c Config) Available() bool {
	_, err := os.Stat(filepath.Join(c.ToolDir, "tools", "scenario.js"))
	return err == nil
}

// Result is what the generator reported.
type Result struct {
	ScenarioFile string
	MissionFile  string
	Output       string
}

// Prepare writes the scenario and builds the mission without starting it.
func Prepare(c Config, s Scenario) (Result, error) {
	return run(c, s, false)
}

// Start builds the mission and starts the battle. The generator returns while
// the game opens.
func Start(c Config, s Scenario) (Result, error) {
	return run(c, s, true)
}

func run(c Config, s Scenario, start bool) (Result, error) {
	out, err := filepath.Abs(c.OutDir)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return Result{}, err
	}
	res := Result{
		ScenarioFile: filepath.Join(out, s.ID+".json"),
		MissionFile:  filepath.Join(out, s.ID+".mispack"),
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return res, err
	}
	if err := os.WriteFile(res.ScenarioFile, data, 0o644); err != nil {
		return res, err
	}

	args := []string{"tools/scenario.js", "build", res.ScenarioFile, "--out", res.MissionFile}
	if start {
		args = append(args, "--start")
	}
	if c.GameDir != "" {
		args = append(args, "--game", c.GameDir)
	}
	cmd := exec.Command("node", args...)
	cmd.Dir = c.ToolDir
	output, runErr := cmd.CombinedOutput()
	res.Output = string(output)
	if runErr != nil {
		return res, generatorError(res.Output, runErr)
	}
	if !strings.Contains(res.Output, "validation: ok") {
		return res, errors.New("MiusMissionMaker did not validate the mission")
	}
	if start && !strings.Contains(res.Output, "battle started") {
		return res, errors.New("MiusMissionMaker did not start the battle")
	}
	return res, nil
}

// generatorError picks the generator's own message from its output.
func generatorError(output string, err error) error {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "VALIDATION FAILED") {
			return errors.New(line)
		}
	}
	return fmt.Errorf("MiusMissionMaker: %w", err)
}
