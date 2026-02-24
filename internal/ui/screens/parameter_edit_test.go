package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/types"
)

func TestParameterEdit_EscapeReturnsBackMsg(t *testing.T) {
	m := NewParameterEdit()

	param := &aws.Parameter{Name: "/test", Type: "String", Value: `{"a":"b"}`}
	_ = m.LoadParameter(param, nil, "a")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected cmd for esc, got nil")
	}

	msg := cmd()
	if _, ok := msg.(types.BackMsg); !ok {
		t.Fatalf("expected types.BackMsg, got %T", msg)
	}
}

