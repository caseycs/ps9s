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
	parameters   []*aws.Parameter
	filtered     []*aws.Parameter
	list         list.Model
	searchInput  textinput.Model
	spinner      spinner.Model
	loading      bool
	searchActive bool
	client       *aws.Client
	err          error
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

// Update handles messages for the parameter list
func (m ParameterListModel) Update(msg tea.Msg) (ParameterListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case types.ParametersLoadedMsg:
		m.parameters = msg.Parameters
		m.filtered = msg.Parameters
		m.loading = false
		m.updateList()
		m.list.Title = fmt.Sprintf("Parameters (%d)", len(m.parameters))
		return m, nil

	case types.ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		h := msg.Height - 4 // Leave space for help text and search
		if m.searchActive {
			h -= 2
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
			case "backspace", "esc":
				return m, func() tea.Msg { return types.BackMsg{} }
			case "q", "ctrl+c":
				return m, tea.Quit
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
		help := "Press 'enter' to view • '/' to search • 'esc' to go back • 'q' to quit"
		if len(m.filtered) != len(m.parameters) {
			help = fmt.Sprintf("Filtered: %d/%d • ", len(m.filtered), len(m.parameters)) + help
		}
		b.WriteString(styles.HelpStyle.Render(help))
	}

	return b.String()
}

// SetSize updates the dimensions of the parameter list
func (m *ParameterListModel) SetSize(width, height int) {
	m.list.SetWidth(width)
	h := height - 4
	if m.searchActive {
		h -= 2
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
	m.list.Title = fmt.Sprintf("Parameters (%d/%d)", len(m.filtered), len(m.parameters))
}

// updateList updates the list items with filtered parameters
func (m *ParameterListModel) updateList() {
	items := make([]list.Item, len(m.filtered))
	for i, p := range m.filtered {
		items[i] = parameterItem{param: p}
	}
	m.list.SetItems(items)
}
