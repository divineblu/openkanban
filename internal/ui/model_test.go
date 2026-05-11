package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/divineblu/openkanban/internal/board"
	"github.com/divineblu/openkanban/internal/config"
	"github.com/divineblu/openkanban/internal/git"
	"github.com/divineblu/openkanban/internal/project"
	"github.com/divineblu/openkanban/internal/terminal"
)

func newBoardTestModel(t *testing.T) (*Model, *project.Project) {
	t.Helper()
	t.Setenv("OPENKANBAN_CONFIG_DIR", t.TempDir())

	cfg := config.DefaultConfig()
	cfg.UI.SidebarVisible = false
	cfg.Defaults.AutoSpawnAgent = false
	theme := cfg.GetTheme()

	registry := &project.ProjectRegistry{Projects: make(map[string]*project.Project)}
	proj := project.NewProject("Test Project", t.TempDir())
	registry.Projects[proj.ID] = proj

	store := project.NewGlobalTicketStore(registry)
	store.AddProject(proj)

	m := &Model{
		config:             cfg,
		theme:              theme,
		colors:             newUIColors(theme),
		titleInput:         textinput.New(),
		descInput:          textarea.New(),
		branchInput:        textinput.New(),
		labelsInput:        textinput.New(),
		projectInput:       textinput.New(),
		settingsInput:      textinput.New(),
		filterInput:        textinput.New(),
		addProjectPath:     textinput.New(),
		blockerFilterInput: textinput.New(),
		globalStore:        store,
		projectRegistry:    registry,
		columns:            board.DefaultColumns(),
		filterProjectIDs:   make(map[string]bool),
		worktreeMgrs:       make(map[string]*git.WorktreeManager),
		mode:               ModeNormal,
		width:              120,
		height:             32,
		hoverColumn:        -1,
		hoverTicket:        -1,
		panes:              make(map[board.TicketID]*terminal.Pane),
		selectedProject:    proj,
	}
	m.refreshColumnTickets()
	return m, proj
}

func TestQuickMoveTicketMovesWithoutWorktreeSetup(t *testing.T) {
	m, proj := newBoardTestModel(t)
	ticket := board.NewTicket("Move me", proj.ID)
	if err := m.globalStore.Add(ticket); err != nil {
		t.Fatalf("add ticket: %v", err)
	}
	m.refreshColumnTickets()

	if _, cmd := m.quickMoveTicket(); cmd != nil {
		t.Fatalf("quickMoveTicket returned unexpected command")
	}

	if ticket.Status != board.StatusInProgress {
		t.Fatalf("ticket status = %q, want %q", ticket.Status, board.StatusInProgress)
	}
	if ticket.WorktreePath != "" {
		t.Fatalf("ticket worktree path = %q, want empty", ticket.WorktreePath)
	}
	if m.activeColumn != 1 || m.activeTicket != 0 {
		t.Fatalf("active selection = column %d ticket %d, want column 1 ticket 0", m.activeColumn, m.activeTicket)
	}
}

func TestQuickMoveTicketAutoSpawnsWhenMovingToInProgress(t *testing.T) {
	m, proj := newBoardTestModel(t)
	m.config.Defaults.AutoSpawnAgent = true
	m.config.Defaults.DefaultAgent = "codex"

	ticket := board.NewTicket("Spawn codex", proj.ID)
	ticket.AgentType = "codex"
	if err := m.globalStore.Add(ticket); err != nil {
		t.Fatalf("add ticket: %v", err)
	}
	m.refreshColumnTickets()

	if _, cmd := m.quickMoveTicket(); cmd == nil {
		t.Fatalf("quickMoveTicket returned nil command")
	}

	if ticket.Status != board.StatusInProgress {
		t.Fatalf("ticket status = %q, want %q", ticket.Status, board.StatusInProgress)
	}
	if m.mode != ModeSpawning {
		t.Fatalf("mode = %q, want %q", m.mode, ModeSpawning)
	}
	if m.spawningTicketID != ticket.ID {
		t.Fatalf("spawningTicketID = %q, want %q", m.spawningTicketID, ticket.ID)
	}
	if m.spawningAgent != "codex" {
		t.Fatalf("spawningAgent = %q, want codex", m.spawningAgent)
	}
}

func TestNormalModeAddAliasOpensCreateTicket(t *testing.T) {
	m, _ := newBoardTestModel(t)

	if _, cmd := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}); cmd == nil {
		t.Fatalf("handleNormalMode(a) returned nil command")
	}

	if m.mode != ModeCreateTicket {
		t.Fatalf("mode = %q, want %q", m.mode, ModeCreateTicket)
	}
	if m.ticketFormField != formFieldTitle {
		t.Fatalf("ticketFormField = %d, want title field", m.ticketFormField)
	}
}

func TestEnterEditsTicketWhenNoAgentIsRunning(t *testing.T) {
	m, proj := newBoardTestModel(t)
	ticket := board.NewTicket("Edit me", proj.ID)
	if err := m.globalStore.Add(ticket); err != nil {
		t.Fatalf("add ticket: %v", err)
	}
	m.refreshColumnTickets()

	m.handleNormalMode(tea.KeyMsg{Type: tea.KeyEnter})

	if m.mode != ModeEditTicket {
		t.Fatalf("mode = %q, want %q", m.mode, ModeEditTicket)
	}
	if m.editingTicketID != ticket.ID {
		t.Fatalf("editingTicketID = %q, want %q", m.editingTicketID, ticket.ID)
	}
}

func TestDoubleClickEditsTicket(t *testing.T) {
	m, proj := newBoardTestModel(t)
	ticket := board.NewTicket("Double click me", proj.ID)
	if err := m.globalStore.Add(ticket); err != nil {
		t.Fatalf("add ticket: %v", err)
	}
	m.refreshColumnTickets()

	m.handleDoubleClick()

	if m.mode != ModeEditTicket {
		t.Fatalf("mode = %q, want %q", m.mode, ModeEditTicket)
	}
	if m.editingTicketID != ticket.ID {
		t.Fatalf("editingTicketID = %q, want %q", m.editingTicketID, ticket.ID)
	}
}

func TestDescriptionPasteMessageInsertsAttachmentMarkdown(t *testing.T) {
	m, _ := newBoardTestModel(t)
	m.mode = ModeCreateTicket
	m.ticketFormField = formFieldTitle
	m.descInput.SetValue("Existing notes")

	if _, cmd := m.handleDescriptionPasteMsg(descriptionPasteMsg{markdown: "![attached image](.openkanban/attachments/image.png)"}); cmd != nil {
		t.Fatalf("handleDescriptionPasteMsg returned unexpected command")
	}

	value := m.descInput.Value()
	if !strings.Contains(value, "Existing notes") || !strings.Contains(value, "![attached image](.openkanban/attachments/image.png)") {
		t.Fatalf("description value = %q, want existing text and attachment markdown", value)
	}
	if m.ticketFormField != formFieldDescription {
		t.Fatalf("ticketFormField = %d, want description field", m.ticketFormField)
	}
}

func TestPasteClipboardImageAttachmentSavesUnderConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("OPENKANBAN_CONFIG_DIR", configDir)

	oldWriter := writeClipboardImageFile
	t.Cleanup(func() {
		writeClipboardImageFile = oldWriter
	})

	var writtenPath string
	writeClipboardImageFile = func(path string) error {
		writtenPath = path
		return os.WriteFile(path, []byte("png data"), 0644)
	}

	proj := project.NewProject("Attachment Project", t.TempDir())
	markdown, err := pasteClipboardImageAttachment(proj, "Screenshot Bug")
	if err != nil {
		t.Fatalf("pasteClipboardImageAttachment error: %v", err)
	}

	wantDir := filepath.Join(configDir, "attachments", board.Slugify(proj.ID, 64))
	if !strings.HasPrefix(writtenPath, wantDir+string(filepath.Separator)) {
		t.Fatalf("writtenPath = %q, want under %q", writtenPath, wantDir)
	}
	if strings.Contains(writtenPath, proj.RepoPath) {
		t.Fatalf("writtenPath = %q, should not be inside project repo %q", writtenPath, proj.RepoPath)
	}
	if _, err := os.Stat(writtenPath); err != nil {
		t.Fatalf("expected attachment file to exist: %v", err)
	}
	if !strings.Contains(markdown, "![attached image](") || !strings.Contains(markdown, filepath.ToSlash(writtenPath)) {
		t.Fatalf("markdown = %q, want attachment markdown with written path", markdown)
	}
}

func TestMouseDragMovesTicketOnReleaseTarget(t *testing.T) {
	m, proj := newBoardTestModel(t)
	ticket := board.NewTicket("Drag me", proj.ID)
	if err := m.globalStore.Add(ticket); err != nil {
		t.Fatalf("add ticket: %v", err)
	}
	m.refreshColumnTickets()

	layouts := m.visibleColumnHitLayouts()
	if len(layouts) < 2 {
		t.Fatalf("expected at least 2 visible columns, got %d", len(layouts))
	}

	sourceX := layouts[0].x + 2
	targetX := layouts[1].x + 2
	ticketY := m.headerHeight() + columnHeaderHeight

	if _, cmd := m.handleMouse(tea.MouseMsg{
		X:      sourceX,
		Y:      ticketY,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}); cmd != nil {
		t.Fatalf("mouse press returned unexpected command")
	}

	if !m.dragging {
		t.Fatalf("expected drag to start")
	}

	if _, cmd := m.handleMouse(tea.MouseMsg{
		X:      targetX,
		Y:      ticketY,
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
	}); cmd != nil {
		t.Fatalf("mouse release returned unexpected command")
	}

	if ticket.Status != board.StatusInProgress {
		t.Fatalf("ticket status = %q, want %q", ticket.Status, board.StatusInProgress)
	}
	if m.dragging {
		t.Fatalf("expected drag to stop")
	}
	if m.activeColumn != 1 || m.activeTicket != 0 {
		t.Fatalf("active selection = column %d ticket %d, want column 1 ticket 0", m.activeColumn, m.activeTicket)
	}
}

func TestRefreshColumnTicketsUsesDeterministicOrder(t *testing.T) {
	m, proj := newBoardTestModel(t)
	base := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)

	low := board.NewTicket("Low", proj.ID)
	low.Priority = 5
	low.CreatedAt = base
	high := board.NewTicket("High", proj.ID)
	high.Priority = 1
	high.CreatedAt = base.Add(time.Minute)
	olderHigh := board.NewTicket("Older High", proj.ID)
	olderHigh.Priority = 1
	olderHigh.CreatedAt = base.Add(-time.Minute)

	for _, ticket := range []*board.Ticket{low, high, olderHigh} {
		if err := m.globalStore.Add(ticket); err != nil {
			t.Fatalf("add ticket: %v", err)
		}
	}

	m.refreshColumnTickets()

	got := m.columnTickets[0]
	if len(got) != 3 {
		t.Fatalf("got %d backlog tickets, want 3", len(got))
	}
	want := []board.TicketID{olderHigh.ID, high.ID, low.ID}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("ticket order[%d] = %q, want %q", i, got[i].ID, want[i])
		}
	}
}
