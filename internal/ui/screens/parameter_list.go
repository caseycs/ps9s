package screens

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilia/ps9s/internal/aws"
	cfg "github.com/ilia/ps9s/internal/config"
	"github.com/ilia/ps9s/internal/styles"
	"github.com/ilia/ps9s/internal/types"
)

// parameterItem represents a parameter in the list
type parameterItem struct {
	param *aws.Parameter
}

func (i parameterItem) FilterValue() string { return i.param.Name }

type paramDelegate struct{}

func (d paramDelegate) Height() int                             { return 1 }
func (d paramDelegate) Spacing() int                            { return 0 }
func (d paramDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d paramDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(parameterItem)
	if !ok {
		return
	}

	var nameStr string
	if index == m.Index() {
		nameStr = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Render("▸ " + i.param.Name)
	} else {
		nameStr = lipgloss.NewStyle().
			PaddingLeft(2).
			Render(i.param.Name)
	}

	fmt.Fprint(w, nameStr)
}

// ParameterListModel represents the parameter list screen
type ParameterListModel struct {
	parameters     []*aws.Parameter
	filtered       []*aws.Parameter
	list           list.Model
	searchInput    textinput.Model
	spinner        spinner.Model
	loading        bool
	searchActive   bool
	client         *aws.Client
	err            error
	currentProfile string
	currentRegion  string
	// Recent profile+region entries (most recent first)
	recents []cfg.RecentEntry
}

// NewParameterList creates a new parameter list screen
func NewParameterList() ParameterListModel {
	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Search parameters..."
	ti.CharLimit = 156

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	const defaultWidth = 80
	const defaultHeight = 20

	l := list.New([]list.Item{}, paramDelegate{}, defaultWidth, defaultHeight)
	l.Title = "Parameters"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = styles.TitleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	return ParameterListModel{
		searchInput: ti,
		spinner:     s,
		list:        l,
	}
}

// Init initializes the parameter list
func (m ParameterListModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// LoadParameters starts loading parameters from AWS
func (m *ParameterListModel) LoadParameters(client *aws.Client) tea.Cmd {
	m.client = client
	m.loading = true
	m.err = nil
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			params, err := client.ListParameters(context.Background())
			if err != nil {
				return types.ErrorMsg{Err: err}
			}
			return types.ParametersLoadedMsg{Parameters: params}
		},
	)
}

// SetRecents updates recent entries shown on the list screen
func (m *ParameterListModel) SetRecents(entries []cfg.RecentEntry) {
	m.recents = entries
}

// Update handles messages for the parameter list
func (m ParameterListModel) Update(msg tea.Msg) (ParameterListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case types.ParametersLoadedMsg:
		m.parameters = msg.Parameters
		m.filtered = msg.Parameters
		m.loading = false
		m.updateList()
		m.updateListTitle()
		return m, nil

	case types.ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		h := msg.Height - 7 // Leave space for help text, search and recents (5 lines)
		if m.searchActive {
			h -= 2
		}
		// Additional space for recents if present
		if len(m.recents) > 0 {
			h -= 7 // 1 label line + 5 recent entries + 1 spacing
		}
		m.list.SetHeight(h)
		return m, nil

	case tea.KeyMsg:
		// Handle search activation
		if msg.String() == "/" && !m.searchActive && !m.loading {
			m.searchActive = true
			m.searchInput.Focus()
			return m, textinput.Blink
		}

		// Handle search mode
		if m.searchActive {
			switch msg.String() {
			case "esc":
				m.searchActive = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.filtered = m.parameters
				m.updateList()
				return m, nil
			case "enter":
				m.searchActive = false
				m.searchInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.filterParameters()
				return m, cmd
			}
		}

		// Regular navigation
		if !m.loading {
			switch msg.String() {
			case "enter":
				// View selected parameter
				if item, ok := m.list.SelectedItem().(parameterItem); ok {
					return m, func() tea.Msg {
						return types.ViewParameterMsg{Parameter: item.param}
					}
				}
			case "e":
				// View selected parameter (shortcut)
				if item, ok := m.list.SelectedItem().(parameterItem); ok {
					return m, func() tea.Msg {
						return types.ViewParameterMsg{Parameter: item.param}
					}
				}
			case "p":
				// Jump to profile selection
				return m, func() tea.Msg { return types.GoToProfileSelectionMsg{} }
			case "backspace", "esc":
				return m, func() tea.Msg { return types.BackMsg{} }
			case "q", "ctrl+c":
				return m, tea.Quit
			case "1", "2", "3", "4", "5":
				// Switch to a recent entry if present
				idx := int(msg.String()[0] - '1')
				if idx >= 0 && idx < len(m.recents) {
					e := m.recents[idx]
					// Don't reload if already on this profile+region
					if e.Profile == m.currentProfile && e.Region == m.currentRegion {
						return m, nil
					}
					return m, func() tea.Msg {
						return types.SwitchRecentMsg{Profile: e.Profile, Region: e.Region}
					}
				}
			}
		}
	}

	// Update spinner if loading
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the parameter list
func (m ParameterListModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading parameters...\n\n", m.spinner.View())
	}

	if m.err != nil {
		return styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n" +
			styles.HelpStyle.Render("Press 'esc' to go back")
	}

	var b strings.Builder

	b.WriteString(m.list.View())
	b.WriteString("\n")

	if m.searchActive {
		b.WriteString("\n")
		b.WriteString(styles.LabelStyle.Render("Search: "))
		b.WriteString(m.searchInput.View())
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("Press 'esc' to cancel search, 'enter' to apply"))
	} else {
		help := "Press 'enter' to view • 'e' to view • '/' to search • 'p' for profile • 'esc' to go back • 'q' to quit"
		if len(m.filtered) != len(m.parameters) {
			help = fmt.Sprintf("Filtered: %d/%d • ", len(m.filtered), len(m.parameters)) + help
		}
		// If we have recent entries, mention keys 1-5
		if len(m.recents) > 0 {
			help += " • press 1-5 to switch recent profile/region"
		}
		b.WriteString(styles.HelpStyle.Render(help))
	}

	// Render recents at bottom (max 5)
	if len(m.recents) > 0 {
		b.WriteString("\n\n")
		b.WriteString(styles.LabelStyle.Render("Recent lists: "))
		b.WriteString("\n")
		for i, r := range m.recents {
			if i >= 5 {
				break
			}
			line := fmt.Sprintf(" %d) %s : %s", i+1, r.Profile, r.Region)
			// Mark current context as inactive
			if r.Profile == m.currentProfile && r.Region == m.currentRegion {
				line = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(line + " (current)")
			}
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

// SetContext sets profile/region context for the list
func (m *ParameterListModel) SetContext(profile, region string) {
	m.currentProfile = profile
	m.currentRegion = region
	m.updateListTitle()
}

// SetSize updates the dimensions of the parameter list
func (m *ParameterListModel) SetSize(width, height int) {
	m.list.SetWidth(width)
	h := height - 7 // Leave space for help text, search and recents (5 lines)
	if m.searchActive {
		h -= 2
	}
	// Additional space for recents if present
	if len(m.recents) > 0 {
		h -= 7 // 1 label line + 5 recent entries + 1 spacing
	}
	m.list.SetHeight(h)
}

// filterParameters filters the parameter list based on search input
func (m *ParameterListModel) filterParameters() {
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.filtered = m.parameters
	} else {
		m.filtered = []*aws.Parameter{}
		for _, p := range m.parameters {
			if strings.Contains(strings.ToLower(p.Name), query) {
				m.filtered = append(m.filtered, p)
			}
		}
	}
	m.updateList()
	m.updateListTitle()
}

// updateList updates the list items with filtered parameters
func (m *ParameterListModel) updateList() {
	items := make([]list.Item, len(m.filtered))
	for i, p := range m.filtered {
		items[i] = parameterItem{param: p}
	}
	m.list.SetItems(items)
}

// updateListTitle updates the title to include profile and region
func (m *ParameterListModel) updateListTitle() {
	// Safe defaults
	profile := m.currentProfile
	region := m.currentRegion
	if profile == "" {
		profile = "-"
	}
	if region == "" {
		region = "-"
	}

	if len(m.filtered) != len(m.parameters) {
		m.list.Title = fmt.Sprintf("%s : %s : Parameters (%d/%d)", profile, region, len(m.filtered), len(m.parameters))
		return
	}

	m.list.Title = fmt.Sprintf("%s : %s : Parameters (%d)", profile, region, len(m.parameters))
}
