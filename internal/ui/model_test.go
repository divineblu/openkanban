package ui

import (
	"testing"
	"time"

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
	theme := cfg.GetTheme()

	registry := &project.ProjectRegistry{Projects: make(map[string]*project.Project)}
	proj := project.NewProject("Test Project", t.TempDir())
	registry.Projects[proj.ID] = proj

	store := project.NewGlobalTicketStore(registry)
	store.AddProject(proj)

	m := &Model{
		config:           cfg,
		theme:            theme,
		colors:           newUIColors(theme),
		globalStore:      store,
		projectRegistry:  registry,
		columns:          board.DefaultColumns(),
		filterProjectIDs: make(map[string]bool),
		worktreeMgrs:     make(map[string]*git.WorktreeManager),
		mode:             ModeNormal,
		width:            120,
		height:           32,
		hoverColumn:      -1,
		hoverTicket:      -1,
		panes:            make(map[board.TicketID]*terminal.Pane),
		selectedProject:  proj,
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
