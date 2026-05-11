package ui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/divineblu/openkanban/internal/board"
	"github.com/divineblu/openkanban/internal/config"
	"github.com/divineblu/openkanban/internal/project"
)

type descriptionPasteMsg struct {
	text     string
	markdown string
	err      error
}

var writeClipboardImageFile = writeClipboardImage

func (m *Model) pasteClipboardToDescription() tea.Cmd {
	proj := m.selectedProject
	if proj == nil {
		if ticket := m.selectedTicket(); ticket != nil {
			proj = m.globalStore.GetProjectForTicket(ticket)
		}
	}
	title := strings.TrimSpace(m.titleInput.Value())

	return func() tea.Msg {
		if markdown, err := pasteClipboardImageAttachment(proj, title); err == nil {
			return descriptionPasteMsg{markdown: markdown}
		}

		text, err := clipboard.ReadAll()
		if err == nil && text != "" {
			return descriptionPasteMsg{text: text}
		}
		if err != nil {
			return descriptionPasteMsg{err: fmt.Errorf("read clipboard: %w", err)}
		}
		return descriptionPasteMsg{err: errors.New("clipboard does not contain text or a supported image")}
	}
}

func (m *Model) handleDescriptionPasteMsg(msg descriptionPasteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.notify("Paste failed: " + msg.err.Error())
		return m, nil
	}
	if m.mode != ModeCreateTicket && m.mode != ModeEditTicket {
		return m, nil
	}

	m.ticketFormField = formFieldDescription
	m.blurAllFormFields()
	m.descInput.Focus()

	switch {
	case msg.markdown != "":
		m.insertDescriptionAttachment(msg.markdown)
		m.notify("Attached image")
	case msg.text != "":
		m.descInput.InsertString(msg.text)
		m.notify("Pasted text")
	}
	return m, nil
}

func (m *Model) insertDescriptionAttachment(markdown string) {
	prefix := ""
	if strings.TrimSpace(m.descInput.Value()) != "" {
		prefix = "\n\n"
	}
	m.descInput.InsertString(prefix + markdown)
}

func pasteClipboardImageAttachment(proj *project.Project, title string) (string, error) {
	configDir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}

	scope := "global"
	if proj != nil && proj.ID != "" {
		scope = board.Slugify(proj.ID, 64)
	}
	dir := filepath.Join(configDir, "attachments", scope)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	slug := board.Slugify(title, 24)
	if slug == "" {
		slug = "ticket"
	}
	filename := fmt.Sprintf("%s-%s.png", time.Now().Format("20060102-150405"), slug)
	absPath := filepath.Join(dir, filename)

	if err := writeClipboardImageFile(absPath); err != nil {
		_ = os.Remove(absPath)
		return "", err
	}

	return fmt.Sprintf("![attached image](%s)", filepath.ToSlash(absPath)), nil
}

func writeClipboardImage(path string) error {
	var failures []string

	if err := writeWithPngpaste(path); err == nil {
		return nil
	} else {
		failures = append(failures, err.Error())
	}

	switch runtime.GOOS {
	case "darwin":
		if err := writeWithMacClipboard(path); err == nil {
			return nil
		} else {
			failures = append(failures, err.Error())
		}
	case "linux":
		if err := writeWithLinuxClipboard(path); err == nil {
			return nil
		} else {
			failures = append(failures, err.Error())
		}
	}

	return fmt.Errorf("image paste unavailable: %s", strings.Join(failures, "; "))
}

func writeWithPngpaste(path string) error {
	bin, err := exec.LookPath("pngpaste")
	if err != nil {
		return errors.New("pngpaste not found")
	}
	cmd := exec.Command(bin, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pngpaste failed: %w%s", err, commandOutputSuffix(output))
	}
	return requireNonEmptyFile(path)
}

func writeWithMacClipboard(path string) error {
	if err := writeMacClipboardClass(path, "PNGf"); err == nil {
		return requireNonEmptyFile(path)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "clipboard-*.tiff")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if err := writeMacClipboardClass(tmpPath, "TIFF"); err != nil {
		return fmt.Errorf("macOS clipboard has no PNG/TIFF image: %w", err)
	}

	bin, err := exec.LookPath("sips")
	if err != nil {
		return errors.New("sips not found for TIFF conversion")
	}
	cmd := exec.Command(bin, "-s", "format", "png", tmpPath, "--out", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sips conversion failed: %w%s", err, commandOutputSuffix(output))
	}
	return requireNonEmptyFile(path)
}

func writeMacClipboardClass(path, className string) error {
	bin, err := exec.LookPath("osascript")
	if err != nil {
		return errors.New("osascript not found")
	}

	classLiteral := "«class " + className + "»"
	lines := []string{
		"set outFile to POSIX file " + appleScriptString(path),
		"set imageData to the clipboard as " + classLiteral,
		"set fileRef to open for access outFile with write permission",
		"set eof of fileRef to 0",
		"write imageData to fileRef",
		"close access fileRef",
	}
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		args = append(args, "-e", line)
	}

	cmd := exec.Command(bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript %s failed: %w%s", className, err, commandOutputSuffix(output))
	}
	return nil
}

func writeWithLinuxClipboard(path string) error {
	if err := writeClipboardCommandOutput(path, "wl-paste", "--no-newline", "--type", "image/png"); err == nil {
		return nil
	}
	if err := writeClipboardCommandOutput(path, "xclip", "-selection", "clipboard", "-t", "image/png", "-o"); err == nil {
		return nil
	}
	return errors.New("wl-paste/xclip image clipboard not available")
}

func writeClipboardCommandOutput(path, command string, args ...string) error {
	bin, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("%s not found", command)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.Command(bin, args...)
	var stderr bytes.Buffer
	cmd.Stdout = file
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%s failed: %w%s", command, err, commandOutputSuffix(stderr.Bytes()))
	}
	return requireNonEmptyFile(path)
}

func requireNonEmptyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return errors.New("clipboard image was empty")
	}
	return nil
}

func commandOutputSuffix(output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return ""
	}
	return ": " + trimmed
}

func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}
