package extended

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunCommand executes sh -c cmd in repoRoot, teeing output to logPath.
func RunCommand(repoRoot, cmd, logPath string) (exitCode int, err error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return 1, err
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return 1, err
	}
	defer func() { _ = logFile.Close() }()

	c := exec.Command("bash", "-lc", cmd)
	c.Dir = repoRoot
	c.Stdout = logFile
	c.Stderr = logFile
	c.Env = os.Environ()
	runErr := c.Run()
	if runErr == nil {
		return 0, nil
	}
	if ee, ok := runErr.(*exec.ExitError); ok {
		return ee.ExitCode(), nil
	}
	return 1, runErr
}

// ValidateRealAppCapture checks ui-real/steps.json declares real_binary.
func ValidateRealAppCapture(runDir string) error {
	path := filepath.Join(runDir, "ui-real", "steps.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read steps.json: %w", err)
	}
	if !strings.Contains(string(data), `"app_mode": "real_binary"`) &&
		!strings.Contains(string(data), `"app_mode":"real_binary"`) {
		return fmt.Errorf("steps.json missing app_mode real_binary")
	}
	return nil
}

// RowPNGPath returns absolute PNG path for a step_ux row under runDir.
func RowPNGPath(runDir string, r Row) string {
	if r.PNG == "" {
		return ""
	}
	return filepath.Join(runDir, filepath.FromSlash(r.PNG))
}

// PNGExistsAndNonEmpty returns false when png missing or tiny.
func PNGExistsAndNonEmpty(path string, minBytes int64) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.Size() >= minBytes
}
