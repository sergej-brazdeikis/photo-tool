//go:build darwin

package gui_macos_test

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-vgo/robotgo"

	"photo-tool/internal/config"
)

func TestMacOSGUIE2E_smokeReviewNav(t *testing.T) {
	if os.Getenv("PHOTO_TOOL_GUI_E2E_MACOS") != "1" {
		t.Skip("set PHOTO_TOOL_GUI_E2E_MACOS=1 to run (requires display, Accessibility, CGO; see tests/gui_macos/README.md)")
	}

	robotgo.Scale = true

	repo := moduleRoot(t)
	lib := t.TempDir()
	if err := config.EnsureLibraryLayout(lib); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(t.TempDir(), uniqueBinaryName())
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = repo
	build.Env = os.Environ()
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	ctx, cancel := testDeadlineContext(t)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = append(os.Environ(),
		config.EnvLibraryRoot+"="+lib,
		"FYNE_SCALE=1",
	)
	var childErr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &childErr)
	cmd.Stderr = io.MultiWriter(os.Stderr, &childErr)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		waitDone := make(chan struct{})
		go func() {
			_, _ = cmd.Process.Wait()
			close(waitDone)
		}()
		select {
		case <-waitDone:
		case <-time.After(4 * time.Second):
			_ = cmd.Process.Kill()
			<-waitDone
		}
	})

	geom, ok := waitWindowGeom(t, pid, 25*time.Second)
	if !ok {
		saveFailureCapture(t, "no-window-bounds")
		t.Logf("child output (tail):\n%s", tailString(childErr.String(), 4000))
		t.Fatal("timed out waiting for GUI window bounds (robotgo and System Events); grant Accessibility to Terminal/Cursor/go and use a logged-in GUI session — see tests/gui_macos/README.md")
	}

	if err := robotgo.ActivePid(pid, 1); err != nil {
		t.Logf("ActivePid: %v", err)
	}
	_ = procSetFrontmost(pid)
	// Let the GL surface and window chrome settle before the baseline screencapture.
	time.Sleep(1200 * time.Millisecond)

	x, y, w, h := geom.x, geom.y, geom.w, geom.h
	if w < 200 || h < 200 {
		saveFailureCapture(t, "tiny-bounds")
		t.Fatalf("unexpected window bounds x=%d y=%d w=%d h=%d", x, y, w, h)
	}

	// Compare the **entire window** PNG bytes (hash) — more sensitive than subsampling the main panel alone.
	capDir := t.TempDir()
	winPath := filepath.Join(capDir, "win.png")
	before, err := captureWindowPNGHash(winPath, x, y, w, h)
	if err != nil {
		t.Fatalf("baseline screencapture: %v (grant Screen Recording to the app that runs go test — see README)", err)
	}

	tryPass := func(tag string) bool {
		t.Helper()
		time.Sleep(750 * time.Millisecond)
		after, err := captureWindowPNGHash(winPath, x, y, w, h)
		if err != nil {
			t.Logf("%s: post screencapture: %v", tag, err)
			return false
		}
		if after != before {
			t.Logf("%s: window capture changed (hash ok)", tag)
			return true
		}
		return false
	}

	railX := x + getenvInt(t, "PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_X", 84)
	offY := getenvInt(t, "PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_Y", 0)

	// 1) Best when exposed: AX — direct press on the Review control.
	if err := clickReviewNavButton(pid); err != nil {
		t.Logf("AX Review click: %v", err)
	} else if tryPass("ax-review") {
		return
	} else {
		t.Logf("AX Review click ran but window hash unchanged")
	}

	// 2) Robotgo mouse (CGO; Retina via robotgo.Scale) — often succeeds for Fyne when the AX tree is sparse.
	for _, dy := range []int{110, 122, 134, 146, 158, 170, 182, 194, 104, 128, 152, 176, 200} {
		clickY := y + dy + offY
		robotgo.Move(railX, clickY)
		robotgo.MilliSleep(40)
		robotgo.Click()
		if tryPass("robotgo-click") {
			return
		}
	}

	// 3) Process-scoped global click (System Events in the target process).
	for _, dy := range []int{98, 110, 122, 134, 146, 158, 170, 182, 194, 206, 218, 104, 128, 152, 176, 200} {
		clickY := y + dy + offY
		if err := procGlobalClick(pid, railX, clickY); err != nil {
			t.Logf("procGlobalClick: %v", err)
			continue
		}
		if tryPass("proc-click") {
			return
		}
	}

	// 4) Bare global click (legacy System Events form).
	for _, dy := range []int{98, 122, 146, 170, 194, 110, 134, 158, 182, 206} {
		clickY := y + dy + offY
		if err := globalClick(railX, clickY); err != nil {
			t.Logf("globalClick: %v", err)
			continue
		}
		if tryPass("global-click") {
			return
		}
	}

	if waitUntilAxContains(pid, "Your library has no photos to show yet", 4*time.Second) ||
		axContains(pid, "Minimum rating") {
		return
	}
	saveFailureCapture(t, "no-review-panel")
	t.Fatal("window capture unchanged after Review navigation attempts; try PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_X / Y, and ensure Screen Recording is enabled for the process that runs go test (Cursor, Terminal, or Cursor Helper) — see tests/gui_macos/README.md")
}

// captureWindowPNGHash writes a PNG of the given screen rect and returns an FNV hash of the **file bytes**
// (any pixel or PNG metadata change alters the hash).
func captureWindowPNGHash(outPath string, x, y, w, h int) (uint64, error) {
	rect := fmt.Sprintf("%d,%d,%d,%d", x, y, w, h)
	cmd := exec.Command("screencapture", "-x", "-R", rect, outPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return 0, err
	}
	sum := fnv.New64a()
	_, _ = sum.Write(data)
	return sum.Sum64(), nil
}

func testDeadlineContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	if dl, ok := t.Deadline(); ok {
		return context.WithDeadline(context.Background(), dl)
	}
	return context.WithTimeout(context.Background(), 10*time.Minute)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for d := dir; d != filepath.Dir(d); d = filepath.Dir(d) {
		data, err := os.ReadFile(filepath.Join(d, "go.mod"))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "module photo-tool") {
			return d
		}
	}
	t.Fatalf("cannot find photo-tool module root from cwd %q", dir)
	return ""
}

func uniqueBinaryName() string {
	return fmt.Sprintf("pt-guie2e-%d", time.Now().UnixNano())
}

func getenvInt(t *testing.T, key string, def int) int {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("%s: %v", key, err)
	}
	return n
}

type winGeom struct {
	x, y, w, h int
}

func waitWindowGeom(t *testing.T, pid int, total time.Duration) (winGeom, bool) {
	t.Helper()
	deadline := time.Now().Add(total)
	for time.Now().Before(deadline) {
		if g, ok := windowGeom(pid); ok {
			return g, true
		}
		robotgo.MilliSleep(400)
	}
	return winGeom{}, false
}

func windowGeom(pid int) (winGeom, bool) {
	x, y, w, h := robotgo.GetBounds(pid, 1)
	if w > 200 && h > 200 {
		return winGeom{x, y, w, h}, true
	}
	if g, ok := windowGeomSystemEvents(pid); ok {
		return g, true
	}
	return winGeom{}, false
}

func globalClick(x, y int) error {
	script := fmt.Sprintf(`tell application "System Events" to click at {%d, %d}`, x, y)
	return exec.Command("osascript", "-e", script).Run()
}

// procGlobalClick brings the app forward then issues a screen-space click in that process context
// (more reliable than a bare global click for some hosts).
func procGlobalClick(pid int, x, y int) error {
	script := fmt.Sprintf(`tell application "System Events"
  tell (first process whose unix id is %d)
    set frontmost to true
    click at {%d, %d}
  end tell
end tell`, pid, x, y)
	return exec.Command("osascript", "-e", script).Run()
}

func clickReviewNavButton(pid int) error {
	// Fyne nests nav buttons; scan the AX tree instead of `button "Review" of window 1`.
	script := fmt.Sprintf(`tell application "System Events"
  tell (first process whose unix id is %d)
    if (count of windows) < 1 then error "no window"
    repeat with win in windows
    repeat with g in entire contents of win
      try
        if class of g is button or class of g is radio button or class of g is pop up button then
          try
            if (value of g as string) is "Review" then
              click g
              return "ok"
            end if
          end try
          try
            if (name of g as string) contains "Review" then
              click g
              return "ok"
            end if
          end try
          try
            if (description of g as string) contains "Review" then
              click g
              return "ok"
            end if
          end try
        end if
      end try
    end repeat
    end repeat
  end tell
  error "Review button not found"
end tell`, pid)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	if strings.TrimSpace(string(out)) != "ok" {
		return fmt.Errorf("unexpected script output: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func procSetFrontmost(pid int) bool {
	script := fmt.Sprintf(`tell application "System Events"
  try
    set frontmost of (first process whose unix id is %d) to true
    return "ok"
  on error
    return "no"
  end try
end tell`, pid)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) == "ok"
}

func axContains(pid int, needle string) bool {
	// Fyne exposes copy in value/name/description on varied AX roles — scan every window.
	script := fmt.Sprintf(`tell application "System Events"
  try
    tell (first process whose unix id is %d)
      if (count of windows) < 1 then return "no"
      repeat with win in windows
        repeat with e in entire contents of win
          try
            if (value of e as string) contains %q then return "yes"
          end try
          try
            if (description of e as string) contains %q then return "yes"
          end try
          try
            if (name of e as string) contains %q then return "yes"
          end try
        end repeat
      end repeat
    end tell
  end try
  return "no"
end tell`, pid, needle, needle, needle)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "yes"
}

func waitUntilAxContains(pid int, needle string, total time.Duration) bool {
	deadline := time.Now().Add(total)
	for time.Now().Before(deadline) {
		if axContains(pid, needle) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

func windowGeomSystemEvents(pid int) (winGeom, bool) {
	script := fmt.Sprintf(`tell application "System Events"
  try
    set proc to first process whose unix id is %d
    tell proc
      if (count of windows) < 1 then return "nowin"
      set p to position of window 1
      set s to size of window 1
      return (item 1 of p as text) & "," & (item 2 of p as text) & "," & (item 1 of s as text) & "," & (item 2 of s as text)
    end tell
  on error errMsg number errNum
    return "error," & errNum & "," & errMsg
  end try
end tell`, pid)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return winGeom{}, false
	}
	line := strings.TrimSpace(string(out))
	if line == "" || strings.HasPrefix(line, "error,") || line == "nowin" {
		return winGeom{}, false
	}
	parts := strings.Split(line, ",")
	if len(parts) != 4 {
		return winGeom{}, false
	}
	var g winGeom
	var ok bool
	g.x, ok = atoiOr(parts[0], 0)
	if !ok {
		return winGeom{}, false
	}
	g.y, ok = atoiOr(parts[1], 0)
	if !ok {
		return winGeom{}, false
	}
	g.w, ok = atoiOr(parts[2], 0)
	if !ok {
		return winGeom{}, false
	}
	g.h, ok = atoiOr(parts[3], 0)
	if !ok {
		return winGeom{}, false
	}
	if g.w < 200 || g.h < 200 {
		return winGeom{}, false
	}
	return g, true
}

func atoiOr(s string, def int) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def, false
	}
	return n, true
}

func tailString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

func saveFailureCapture(t *testing.T, tag string) {
	t.Helper()
	bundle := strings.TrimSpace(os.Getenv("PHOTO_TOOL_GUI_E2E_BUNDLE"))
	if bundle == "" {
		return
	}
	_ = os.MkdirAll(filepath.Join(bundle, "logs"), 0o755)
	path := filepath.Join(bundle, "logs", "gui-e2e-failure-"+tag+".png")
	if err := exec.Command("screencapture", "-x", path).Run(); err != nil {
		t.Logf("screencapture debug: %v", err)
		return
	}
	t.Logf("wrote failure screenshot %s", path)
}
