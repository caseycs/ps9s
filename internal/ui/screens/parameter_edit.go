package screens

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/styles"
	"github.com/ilia/ps9s/internal/types"
)

// ParameterEditModel represents the parameter edit screen
type ParameterEditModel struct {
	parameter      *aws.Parameter
	client         *aws.Client
	isJSON         bool
	jsonData       map[string]interface{} // Parsed JSON
	textarea       textarea.Model         // Value editor
	selectedKey    string                 // Currently selected key path
	spinner        spinner.Model
	saving         bool
	err            error
	width          int
	height         int
	currentProfile string
	currentRegion  string
}

// NewParameterEdit creates a new parameter edit screen
func NewParameterEdit() ParameterEditModel {
	ta := textarea.New()
	ta.Placeholder = "Enter parameter value..."
	ta.CharLimit = 0
	ta.ShowLineNumbers = false

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return ParameterEditModel{
		textarea: ta,
		spinner:  s,
	}
}

// Init initializes the parameter edit screen
func (m ParameterEditModel) Init() tea.Cmd {
	return textarea.Blink
}

// LoadParameter loads a parameter for editing
func (m *ParameterEditModel) LoadParameter(param *aws.Parameter, client *aws.Client, jsonKey string) tea.Cmd {
	m.parameter = param
	m.client = client
	m.err = nil
	m.saving = false
	m.selectedKey = jsonKey

	// Check if value is JSON
	m.isJSON = isValidJSON(param.Value)

	if m.isJSON && jsonKey != "" {
		// Editing a specific JSON key
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(param.Value), &data); err == nil {
			m.jsonData = data

			// Find the value for the specified key
			value := m.getJSONValue(data, jsonKey)
			m.textarea.SetValue(value)
			m.textarea.Focus()
		} else {
			// JSON parsing failed, fall back to raw edit
			m.isJSON = false
			m.textarea.SetValue(param.Value)
			m.textarea.Focus()
		}
	} else {
		// Not JSON or no key specified, edit raw value
		m.isJSON = false
		m.textarea.SetValue(param.Value)
		m.textarea.Focus()
	}

	return textarea.Blink
}

// getJSONValue retrieves a value from JSON using dot notation path
func (m *ParameterEditModel) getJSONValue(data interface{}, path string) string {
	parts := m.parsePath(path)
	if len(parts) == 0 {
		return ""
	}

	current := data
	for _, part := range parts {
		if part.isArray {
			arr, ok := current.([]interface{})
			if !ok || part.index >= len(arr) {
				return ""
			}
			current = arr[part.index]
		} else {
			obj, ok := current.(map[string]interface{})
			if !ok {
				return ""
			}
			val, exists := obj[part.key]
			if !exists {
				return ""
			}
			current = val
		}
	}

	// Convert final value to string
	switch v := current.(type) {
	case string:
		return v
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Update handles messages for the parameter edit screen
func (m ParameterEditModel) Update(msg tea.Msg) (ParameterEditModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(msg.Height - 10)
		return m, nil

	case types.SaveSuccessMsg:
		m.saving = false
		// Go back to view screen
		return m, func() tea.Msg { return types.BackMsg{} }

	case types.ErrorMsg:
		m.saving = false
		m.err = msg.Err
		return m, nil

	case tea.KeyMsg:
		if m.saving {
			return m, nil
		}

		// Handle edit mode keys
		switch msg.String() {
		case "esc":
			// Go back to view screen
			return m, func() tea.Msg { return types.BackMsg{} }
		case "ctrl+s":
			// Save the value
			return m, m.saveParameter()
		case "ctrl+c":
			return m, tea.Quit
		}

		// Update textarea
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
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

// saveParameter saves the edited parameter value
func (m *ParameterEditModel) saveParameter() tea.Cmd {
	m.saving = true
	m.err = nil

	newValue := m.textarea.Value()

	// If editing JSON key, reconstruct the JSON
	if m.isJSON && m.selectedKey != "" {
		if err := m.updateJSONValue(m.jsonData, m.selectedKey, newValue); err != nil {
			return func() tea.Msg {
				return types.ErrorMsg{Err: fmt.Errorf("failed to update JSON: %w", err)}
			}
		}

		// Marshal back to JSON
		jsonBytes, err := json.MarshalIndent(m.jsonData, "", "  ")
		if err != nil {
			return func() tea.Msg {
				return types.ErrorMsg{Err: fmt.Errorf("failed to marshal JSON: %w", err)}
			}
		}
		newValue = string(jsonBytes)
	}

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

// updateJSONValue updates a value in nested JSON structure using dot notation path
func (m *ParameterEditModel) updateJSONValue(data interface{}, path string, newValue string) error {
	// Parse path (e.g., "server.host" or "items[0].name")
	parts := m.parsePath(path)

	if len(parts) == 0 {
		return fmt.Errorf("invalid path: %s", path)
	}

	// Navigate to parent
	current := data
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		if part.isArray {
			arr, ok := current.([]interface{})
			if !ok {
				return fmt.Errorf("expected array at %s", part.key)
			}
			if part.index >= len(arr) {
				return fmt.Errorf("index out of range at %s", part.key)
			}
			current = arr[part.index]
		} else {
			obj, ok := current.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected object at %s", part.key)
			}
			val, exists := obj[part.key]
			if !exists {
				return fmt.Errorf("key not found: %s", part.key)
			}
			current = val
		}
	}

	// Update the final value
	lastPart := parts[len(parts)-1]

	// Try to parse newValue as appropriate type
	var parsedValue interface{}
	parsedValue = newValue // default to string

	// Try to detect and parse the type
	if newValue == "null" {
		parsedValue = nil
	} else if newValue == "true" {
		parsedValue = true
	} else if newValue == "false" {
		parsedValue = false
	} else if num := parseNumber(newValue); num != nil {
		parsedValue = num
	}

	if lastPart.isArray {
		arr, ok := current.([]interface{})
		if !ok {
			return fmt.Errorf("expected array at final position")
		}
		if lastPart.index >= len(arr) {
			return fmt.Errorf("index out of range at final position")
		}
		arr[lastPart.index] = parsedValue
	} else {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected object at final position")
		}
		obj[lastPart.key] = parsedValue
	}

	return nil
}

// pathPart represents a part of a JSON path
type pathPart struct {
	key     string
	isArray bool
	index   int
}

// parsePath parses a dot notation path with array indices
func (m *ParameterEditModel) parsePath(path string) []pathPart {
	var parts []pathPart
	current := ""

	for i := 0; i < len(path); i++ {
		ch := path[i]

		switch ch {
		case '.':
			if current != "" {
				parts = append(parts, pathPart{key: current, isArray: false})
				current = ""
			}
		case '[':
			if current != "" {
				// This is an array access
				endBracket := strings.Index(path[i:], "]")
				if endBracket == -1 {
					return nil // Invalid path
				}
				indexStr := path[i+1 : i+endBracket]
				var index int
				fmt.Sscanf(indexStr, "%d", &index)
				parts = append(parts, pathPart{key: current, isArray: true, index: index})
				current = ""
				i += endBracket // Skip to after ]
			}
		case ']':
			// Skip, handled above
		default:
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, pathPart{key: current, isArray: false})
	}

	return parts
}

// parseNumber attempts to parse a string as a number
func parseNumber(s string) interface{} {
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		// Check if it's an integer
		if float64(int64(f)) == f {
			return int64(f)
		}
		return f
	}
	return nil
}

// View renders the parameter edit screen
func (m ParameterEditModel) View() string {
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

	// Show value editor
	if m.isJSON && m.selectedKey != "" {
		b.WriteString("  " + styles.LabelStyle.Render("Editing: "))
		b.WriteString(m.selectedKey)
		b.WriteString("\n\n")
	} else {
		b.WriteString("  " + styles.LabelStyle.Render("Edit Value:"))
		b.WriteString("\n\n")
	}

	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")

	helpText := "Press 'ctrl+s' to save • 'esc' to cancel • 'ctrl+c' to quit"
	b.WriteString("  " + styles.HelpStyle.Render(helpText))

	return b.String()
}

// SetContext sets the profile and region context for the edit screen
func (m *ParameterEditModel) SetContext(profile, region string) {
	m.currentProfile = profile
	m.currentRegion = region
}

// SetSize updates the dimensions of the parameter edit screen
func (m *ParameterEditModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(height - 10)
}
