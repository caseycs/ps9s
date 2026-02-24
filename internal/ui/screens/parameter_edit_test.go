package screens

import (
	"reflect"
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

func TestParsePath_SimpleKey(t *testing.T) {
	m := NewParameterEdit()
	got := m.parsePath("host")
	want := []pathPart{{key: "host"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePath(\"host\") = %+v, want %+v", got, want)
	}
}

func TestParsePath_DottedKey(t *testing.T) {
	m := NewParameterEdit()
	got := m.parsePath("server.host")
	want := []pathPart{{key: "server"}, {key: "host"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePath(\"server.host\") = %+v, want %+v", got, want)
	}
}

func TestParsePath_ArrayIndex(t *testing.T) {
	m := NewParameterEdit()
	got := m.parsePath("items[0]")
	want := []pathPart{{key: "items"}, {isArray: true, index: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePath(\"items[0]\") = %+v, want %+v", got, want)
	}
}

func TestParsePath_ArrayThenKey(t *testing.T) {
	m := NewParameterEdit()
	got := m.parsePath("items[2].name")
	want := []pathPart{
		{key: "items"},
		{isArray: true, index: 2},
		{key: "name"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePath(\"items[2].name\") = %+v, want %+v", got, want)
	}
}

func TestParsePath_NestedArrays(t *testing.T) {
	m := NewParameterEdit()
	got := m.parsePath("a[0].b[1].c")
	want := []pathPart{
		{key: "a"},
		{isArray: true, index: 0},
		{key: "b"},
		{isArray: true, index: 1},
		{key: "c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePath(\"a[0].b[1].c\") = %+v, want %+v", got, want)
	}
}

func TestParsePath_InvalidMissingBracket(t *testing.T) {
	m := NewParameterEdit()
	got := m.parsePath("items[0")
	if got != nil {
		t.Fatalf("expected nil for invalid path, got %+v", got)
	}
}

func TestGetJSONValue_NestedArray(t *testing.T) {
	m := NewParameterEdit()
	param := &aws.Parameter{
		Name:  "/test",
		Type:  "String",
		Value: `{"items":[{"name":"first"},{"name":"second"}]}`,
	}
	_ = m.LoadParameter(param, nil, "items[1].name")

	if m.textarea.Value() != "second" {
		t.Fatalf("expected textarea value \"second\", got %q", m.textarea.Value())
	}
}

func TestGetJSONValue_TopLevelArray(t *testing.T) {
	m := NewParameterEdit()
	param := &aws.Parameter{
		Name:  "/test",
		Type:  "String",
		Value: `{"tags":["alpha","beta","gamma"]}`,
	}
	_ = m.LoadParameter(param, nil, "tags[2]")

	if m.textarea.Value() != "gamma" {
		t.Fatalf("expected textarea value \"gamma\", got %q", m.textarea.Value())
	}
}

