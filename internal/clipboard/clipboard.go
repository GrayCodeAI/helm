// Package clipboard provides cross-platform clipboard integration.
package clipboard

import (
	"os/exec"
	"runtime"
	"strings"
)

// Read reads from the system clipboard
func Read() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbpaste")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	case "linux":
		cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
		out, err := cmd.Output()
		if err != nil {
			// Try xsel
			cmd = exec.Command("xsel", "--clipboard", "--output")
			out, err = cmd.Output()
			if err != nil {
				return "", err
			}
		}
		return string(out), nil
	case "windows":
		cmd := exec.Command("powershell", "-command", "Get-Clipboard")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return "", nil
	}
}

// Write writes to the system clipboard
func Write(text string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "linux":
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		err := cmd.Run()
		if err != nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
		return err
	case "windows":
		cmd := exec.Command("powershell", "-command", "Set-Clipboard", text)
		return cmd.Run()
	default:
		return nil
	}
}
