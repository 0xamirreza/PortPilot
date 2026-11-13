package privilege

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Call runs one helper operation. It asks pkexec for authorization and does not keep root.
func Call(msg Message) (Result, error) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return Result{}, err
	}
	exe, err := os.Executable()
	if err != nil {
		return Result{}, err
	}
	if _, err := exec.LookPath("pkexec"); err != nil {
		return Result{}, fmt.Errorf("pkexec is not installed; the helper was not started")
	}

	var cmd *exec.Cmd
	// When running inside an AppImage, $APPIMAGE contains the host path to the .AppImage file.
	// Executing the AppImage directly with --appimage-extract-and-run allows root to run
	// the helper without encountering FUSE user-mount access restrictions.
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		cmd = exec.Command("pkexec", appimage, "--appimage-extract-and-run", "__privileged")
	} else {
		cmd = exec.Command("pkexec", exe, "__privileged")
	}
	cmd.Stdin = bytes.NewReader(raw)
	out, err := cmd.Output()
	if len(out) == 0 && err != nil {
		return Result{}, fmt.Errorf("authorization failed: %w", err)
	}
	var res Result
	if jerr := json.Unmarshal(out, &res); jerr != nil {
		if err != nil {
			return Result{}, fmt.Errorf("authorization failed: %w", err)
		}
		return Result{}, fmt.Errorf("helper response: %w", jerr)
	}
	if !res.OK {
		if res.Error == "" {
			res.Error = "the helper rejected the operation"
		}
		return res, fmt.Errorf("%s", res.Error)
	}
	return res, nil
}
