package screens

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/styles"
	"github.com/ilia/ps9s/internal/types"
)

// JSONAddModel represents the screen for adding a new JSON key-value pair
type JSONAddModel struct {
	parameter      *aws.Parameter
	client         *aws.Client
	keyInput       textinput.Model
	valueInput     textarea.Model
	focusedInput   int // 0 = key, 1 = value
	spinner        spinner.Model
	saving         bool
	err            error
	width          int
	height         int
	currentProfile string
	currentRegion  string
}

// NewJSONAdd creates a new JSON add screen
func NewJSONAdd() JSONAddModel {
	keyInput := textinput.New()
	keyInput.Placeholder = "Enter key name..."
	keyInput.CharLimit = 256
	keyInput.Width = 60

	valueInput := textarea.New()
	valueInput.Placeholder = "Enter value..."
	valueInput.CharLimit = 0
	valueInput.ShowLineNumbers = false

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return JSONAddModel{
		keyInput:     keyInput,
		valueInput:   valueInput,
		focusedInput: 0,
		spinner:      s,
	}
}

// Init initializes the JSON add screen
func (m JSONAddModel) Init() tea.Cmd {
	return textarea.Blink
}

// LoadParameter loads the parameter to add a JSON key to
func (m *JSONAddModel) LoadParameter(param *aws.Parameter, client *aws.Client) tea.Cmd {
	m.parameter = param
	m.client = client
	m.err = nil
	m.saving = false
	m.focusedInput = 0

	// Reset inputs
	m.keyInput.SetValue("")
	m.valueInput.SetValue("")
	m.keyInput.Focus()
	m.valueInput.Blur()

	return textinput.Blink
}

// Update handles messages for the JSON add screen
func (m JSONAddModel) Update(msg tea.Msg) (JSONAddModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.keyInput.Width = msg.Width - 20
		m.valueInput.SetWidth(msg.Width - 4)
		m.valueInput.SetHeight(msg.Height - 14)
		return m, nil

	case types.ErrorMsg:
		m.saving = false
		m.err = msg.Err
		return m, nil

	case tea.KeyMsg:
		if m.saving {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+s":
			// Validate and save
			if m.keyInput.Value() == "" {
				m.err = fmt.Errorf("key cannot be empty")
				return m, nil
			}
			return m, m.saveNewKey()
		case "esc":
			return m, func() tea.Msg { return types.BackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			// Switch focus between inputs
			if m.focusedInput == 0 {
				m.focusedInput = 1
				m.keyInput.Blur()
				m.valueInput.Focus()
				return m, textarea.Blink
			} else {
				m.focusedInput = 0
				m.valueInput.Blur()
				m.keyInput.Focus()
				return m, textinput.Blink
			}
		case "shift+tab":
			// Switch focus in reverse
			if m.focusedInput == 1 {
				m.focusedInput = 0
				m.valueInput.Blur()
				m.keyInput.Focus()
				return m, textinput.Blink
			} else {
				m.focusedInput = 1
				m.keyInput.Blur()
				m.valueInput.Focus()
				return m, textarea.Blink
			}
		}

		// Update the focused input
		var cmd tea.Cmd
		if m.focusedInput == 0 {
			m.keyInput, cmd = m.keyInput.Update(msg)
		} else {
			m.valueInput, cmd = m.valueInput.Update(msg)
		}
		return m, cmd
	}

	// Update spinner if saving
	if m.saving {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// saveNewKey saves the new key-value pair to the JSON parameter
func (m *JSONAddModel) saveNewKey() tea.Cmd {
	m.saving = true
	m.err = nil

	key := m.keyInput.Value()
	value := m.valueInput.Value()

	// Parse existing JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(m.parameter.Value), &data); err != nil {
		return func() tea.Msg {
			return types.ErrorMsg{Err: fmt.Errorf("failed to parse JSON: %w", err)}
		}
	}

	// Check if key already exists
	if _, exists := data[key]; exists {
		return func() tea.Msg {
			return types.ErrorMsg{Err: fmt.Errorf("key '%s' already exists", key)}
		}
	}

	// Add new key-value pair
	// Try to parse value as appropriate type
	var parsedValue interface{}
	parsedValue = value // default to string

	if value == "null" {
		parsedValue = nil
	} else if value == "true" {
		parsedValue = true
	} else if value == "false" {
		parsedValue = false
	} else if num := parseNumber(value); num != nil {
		parsedValue = num
	}

	data[key] = parsedValue

	// Marshal back to JSON
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return func() tea.Msg {
			return types.ErrorMsg{Err: fmt.Errorf("failed to marshal JSON: %w", err)}
		}
	}
	newValue := string(jsonBytes)

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := m.client.PutParameter(
				context.Background(),
				m.parameter.Name,
				newValue,
				m.parameter.Type,
			)
			if err != nil {
				return types.ErrorMsg{Err: err}
			}
			// Update the parameter with new value
			updatedParam := *m.parameter
			updatedParam.Value = newValue
			return types.SaveSuccessMsg{Parameter: &updatedParam}
		},
	)
}

// View renders the JSON add screen
func (m JSONAddModel) View() string {
	if m.saving {
		return fmt.Sprintf("\n  %s Saving parameter...\n", m.spinner.View())
	}

	var b strings.Builder

	if m.parameter != nil {
		profile := m.currentProfile
		region := m.currentRegion
		if profile == "" {
			profile = "-"
		}
		if region == "" {
			region = "-"
		}
		title := fmt.Sprintf("%s : %s : %s", profile, region, m.parameter.Name)
		b.WriteString("  " + styles.TitleStyle.Render(title))
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString("  " + styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Key input
	b.WriteString("  " + styles.LabelStyle.Render("Key:"))
	b.WriteString("\n\n")
	b.WriteString("  " + m.keyInput.View())
	b.WriteString("\n\n")

	// Value input (textarea)
	b.WriteString("  " + styles.LabelStyle.Render("Value:"))
	b.WriteString("\n\n")
	b.WriteString(m.valueInput.View())
	b.WriteString("\n\n")

	helpText := "tab: switch field • ctrl+s: save • esc: cancel • ctrl+c: quit"
	b.WriteString("  " + styles.HelpStyle.Render(helpText))

	return b.String()
}

// SetContext sets the profile and region context for the add screen
func (m *JSONAddModel) SetContext(profile, region string) {
	m.currentProfile = profile
	m.currentRegion = region
}

// SetSize updates the dimensions of the JSON add screen
func (m *JSONAddModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.keyInput.Width = width - 20
	m.valueInput.SetWidth(width - 4)
	m.valueInput.SetHeight(height - 14)
}
