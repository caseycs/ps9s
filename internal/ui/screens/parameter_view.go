package screens

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/styles"
	"github.com/ilia/ps9s/internal/types"
)

// jsonKeyItem represents a JSON key in the list
type jsonKeyItem struct {
	key   string
	value string
}

func (i jsonKeyItem) FilterValue() string { return i.key }

// clearStatusMsg is used internally to clear transient status messages
type clearStatusMsg struct{}

// copyResultMsg is sent from the async copy command to report result
type copyResultMsg struct {
	Err  error
	Text string
}

// ParameterViewModel represents the parameter view screen
type ParameterViewModel struct {
	parameter      *aws.Parameter
	client         *aws.Client
	viewport       viewport.Model
	spinner        spinner.Model
	loading        bool
	ready          bool
	err            error
	status         string
	isJSON         bool
	jsonKeys       []jsonKeyItem
	currentProfile string
	currentRegion  string
	selectedIndex  int
}

// SetContext sets the profile and region context for the view screen
func (m *ParameterViewModel) SetContext(profile, region string) {
	m.currentProfile = profile
	m.currentRegion = region
}

// NewParameterView creates a new parameter view screen
func NewParameterView() ParameterViewModel {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Padding(1, 2)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return ParameterViewModel{
		viewport: vp,
		spinner:  s,
	}
}

// Init initializes the parameter view
func (m ParameterViewModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// LoadParameter loads a parameter for viewing (fetches full details with value)
func (m *ParameterViewModel) LoadParameter(param *aws.Parameter, client *aws.Client) tea.Cmd {
	m.client = client
	m.parameter = param
	m.loading = true
	m.err = nil
	m.status = ""

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			// Fetch full parameter with decrypted value
			fullParam, err := client.GetParameter(context.Background(), param.Name)
			if err != nil {
				return types.ErrorMsg{Err: err}
			}
			return types.ParameterValueLoadedMsg{Parameter: fullParam}
		},
	)
}

// Update handles messages for the parameter view
func (m ParameterViewModel) Update(msg tea.Msg) (ParameterViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case types.ParameterValueLoadedMsg:
		m.parameter = msg.Parameter
		m.loading = false
		m.selectedIndex = 0

		// Check if value is JSON
		m.isJSON = isValidJSON(msg.Parameter.Value)
		if m.isJSON {
			var data interface{}
			if err := json.Unmarshal([]byte(msg.Parameter.Value), &data); err == nil {
				m.jsonKeys = m.flattenJSONForView(data, "")
			}
		}

		content := m.formatParameterDetails(msg.Parameter)
		m.viewport.SetContent(content)
		return m, nil

	case types.ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case copyResultMsg:
		if msg.Err != nil {
			m.status = fmt.Sprintf("Copy failed: %v", msg.Err)
		} else {
			m.status = "Copied to clipboard"
		}
		return m, nil

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-10)
			m.viewport.Style = lipgloss.NewStyle().Padding(1, 2)
			if m.parameter != nil {
				m.viewport.SetContent(m.formatParameterDetails(m.parameter))
			}
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 10
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "e":
			// Edit parameter or selected JSON key
			if m.isJSON && len(m.jsonKeys) > 0 {
				// Edit selected JSON key
				selectedKey := m.jsonKeys[m.selectedIndex].key
				return m, func() tea.Msg {
					return types.EditParameterMsg{
						Parameter: m.parameter,
						JSONKey:   selectedKey,
					}
				}
			} else {
				// Edit entire parameter value
				return m, func() tea.Msg {
					return types.EditParameterMsg{Parameter: m.parameter}
				}
			}
		case "c":
			// Copy selected value (either JSON key value or whole parameter)
			if m.parameter == nil {
				return m, nil
			}
			var toCopy string
			if m.isJSON && len(m.jsonKeys) > 0 {
				toCopy = m.jsonKeys[m.selectedIndex].value
			} else {
				toCopy = m.parameter.Value
			}

			// Async copy command + delayed clear status
			copyCmd := func() tea.Msg {
				err := clipboard.WriteAll(toCopy)
				return copyResultMsg{Err: err, Text: toCopy}
			}

			clearCmd := func() tea.Msg {
				time.Sleep(2 * time.Second)
				return clearStatusMsg{}
			}

			return m, tea.Batch(copyCmd, clearCmd)
		case "up", "k":
			if m.isJSON && len(m.jsonKeys) > 0 {
				if m.selectedIndex > 0 {
					m.selectedIndex--
					m.viewport.SetContent(m.formatParameterDetails(m.parameter))
				}
				return m, nil
			}
		case "down", "j":
			if m.isJSON && len(m.jsonKeys) > 0 {
				if m.selectedIndex < len(m.jsonKeys)-1 {
					m.selectedIndex++
					m.viewport.SetContent(m.formatParameterDetails(m.parameter))
				}
				return m, nil
			}
		case "backspace", "esc":
			return m, func() tea.Msg { return types.BackMsg{} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	// Update spinner if loading
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the parameter view
func (m ParameterViewModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading parameter value...\n", m.spinner.View())
	}

	if m.err != nil {
		return styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n" +
			styles.HelpStyle.Render("Press 'esc' to go back")
	}

	if m.parameter == nil {
		return "No parameter selected"
	}

	var b strings.Builder

	// Build title with profile and region
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
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	helpText := "Press 'e' to edit"
	if m.isJSON && len(m.jsonKeys) > 0 {
		helpText += " selected key • ↑/↓ to select"
	}
	helpText += " • 'c' to copy • 'esc' to go back • 'q' to quit"
	b.WriteString("  " + styles.HelpStyle.Render(helpText))

	// Always reserve a line for status message
	b.WriteString("\n")
	if m.status != "" {
		b.WriteString("  " + styles.LabelStyle.Render(m.status))
	}

	return b.String()
}

// SetSize updates the dimensions of the parameter view
func (m *ParameterViewModel) SetSize(width, height int) {
	m.viewport.Width = width - 4
	m.viewport.Height = height - 10
}

// isValidJSON checks if a string is valid JSON
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// flattenJSONForView flattens JSON for viewing with selection
func (m *ParameterViewModel) flattenJSONForView(data interface{}, prefix string) []jsonKeyItem {
	var result []jsonKeyItem

	switch v := data.(type) {
	case map[string]interface{}:
		// Sort keys for consistent output
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := v[key]
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			result = append(result, m.flattenJSONForView(value, newPrefix)...)
		}
	case []interface{}:
		for i, value := range v {
			newPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			result = append(result, m.flattenJSONForView(value, newPrefix)...)
		}
	default:
		// Leaf node
		var valueStr string
		switch val := v.(type) {
		case string:
			valueStr = val
		case nil:
			valueStr = "null"
		default:
			valueStr = fmt.Sprintf("%v", val)
		}
		result = append(result, jsonKeyItem{key: prefix, value: valueStr})
	}

	return result
}

// flattenJSON flattens a JSON structure into dot notation key-value pairs
func flattenJSON(data interface{}, prefix string) []string {
	var result []string

	switch v := data.(type) {
	case map[string]interface{}:
		// Sort keys for consistent output
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := v[key]
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			result = append(result, flattenJSON(value, newPrefix)...)
		}
	case []interface{}:
		for i, value := range v {
			newPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			result = append(result, flattenJSON(value, newPrefix)...)
		}
	default:
		// Leaf node - format as key: value
		var valueStr string
		switch val := v.(type) {
		case string:
			valueStr = val
		case nil:
			valueStr = "null"
		default:
			valueStr = fmt.Sprintf("%v", val)
		}
		result = append(result, fmt.Sprintf("%s: %s", prefix, valueStr))
	}

	return result
}

// formatParameterDetails formats the parameter details for display
func (m ParameterViewModel) formatParameterDetails(p *aws.Parameter) string {
	var b strings.Builder

	b.WriteString(styles.LabelStyle.Render("Type: "))
	b.WriteString(p.Type)
	b.WriteString("\n\n")

	b.WriteString(styles.LabelStyle.Render("Value:"))
	b.WriteString("\n\n")

	// Check if value is valid JSON and format accordingly
	var valueContent string
	if m.isJSON && len(m.jsonKeys) > 0 {
		// Display JSON with selection highlighting
		var lines []string
		for i, item := range m.jsonKeys {
			line := fmt.Sprintf("%s: %s", item.key, item.value)
			if i == m.selectedIndex {
				// Highlight selected line
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color("86")).
					Bold(true).
					Render("▸ " + line)
			} else {
				line = "  " + line
			}
			lines = append(lines, line)
		}
		valueContent = strings.Join(lines, "\n")
	} else {
		// Not JSON, display as-is
		valueContent = p.Value
	}

	// Display value in a styled box
	valueBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(m.viewport.Width - 6).
		Render(valueContent)

	b.WriteString(valueBox)

	return b.String()
}
