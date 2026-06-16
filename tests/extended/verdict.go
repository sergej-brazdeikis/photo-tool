package extended

import (
	"bufio"
	"os"
	"strings"
)

// LastLineMachineResult reads the last non-empty line of path and returns it trimmed.
func LastLineMachineResult(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	var last string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line != "" {
			last = line
		}
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return last, nil
}

// ParseStepUXResult returns true when verdict ends with STEP_UX_RESULT=pass.
func ParseStepUXResult(path string) (bool, error) {
	line, err := LastLineMachineResult(path)
	if err != nil {
		return false, err
	}
	return line == "STEP_UX_RESULT=pass", nil
}

// ParseFlowUXResult returns true when verdict ends with FLOW_UX_RESULT=pass.
func ParseFlowUXResult(path string) (bool, error) {
	line, err := LastLineMachineResult(path)
	if err != nil {
		return false, err
	}
	return line == "FLOW_UX_RESULT=pass", nil
}
