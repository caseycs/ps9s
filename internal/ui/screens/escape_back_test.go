package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilia/ps9s/internal/types"
)

func TestProfileSelector_EscapeReturnsBackMsg(t *testing.T) {
	m := NewProfileSelector([]string{"prod"})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected cmd for esc, got nil")
	}
	if _, ok := cmd().(types.BackMsg); !ok {
		t.Fatalf("expected types.BackMsg")
	}
}

func TestRegionSelector_EscapeReturnsBackMsg(t *testing.T) {
	m := NewRegionSelector()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected cmd for esc, got nil")
	}
	if _, ok := cmd().(types.BackMsg); !ok {
		t.Fatalf("expected types.BackMsg")
	}
}

func TestParameterList_EscapeReturnsBackMsg(t *testing.T) {
	m := NewParameterList()
	m.loading = false
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected cmd for esc, got nil")
	}
	if _, ok := cmd().(types.BackMsg); !ok {
		t.Fatalf("expected types.BackMsg")
	}
}

func TestParameterView_EscapeReturnsBackMsg(t *testing.T) {
	m := NewParameterView()
	m.loading = false
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected cmd for esc, got nil")
	}
	if _, ok := cmd().(types.BackMsg); !ok {
		t.Fatalf("expected types.BackMsg")
	}
}
