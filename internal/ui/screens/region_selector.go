package screens

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilia/ps9s/internal/styles"
	"github.com/ilia/ps9s/internal/types"
)

// Common AWS regions
var defaultRegions = []string{
	"eu-central-1",
	"us-east-1",
	"us-west-2",
	"ap-southeast-1",
	"ap-northeast-1",
	"eu-west-1",
	"us-west-1",
	"ap-south-1",
}

// regionItem represents a region in the list
type regionItem struct {
	region string
}

func (i regionItem) FilterValue() string { return i.region }

type regionDelegate struct{}

func (d regionDelegate) Height() int                             { return 1 }
func (d regionDelegate) Spacing() int                            { return 0 }
func (d regionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d regionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(regionItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.region)

	fn := lipgloss.NewStyle().PaddingLeft(2).Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true).
				PaddingLeft(2).
				Render("▸ " + s[0])
		}
	}

	fmt.Fprint(w, fn(str))
}

// RegionSelectorModel represents the region selection screen
type RegionSelectorModel struct {
	list   list.Model
	choice string
}

// NewRegionSelector creates a new region selector screen
func NewRegionSelector() RegionSelectorModel {
	items := make([]list.Item, len(defaultRegions))
	for i, r := range defaultRegions {
		items[i] = regionItem{region: r}
	}

	const defaultWidth = 80
	const defaultHeight = 20

	l := list.New(items, regionDelegate{}, defaultWidth, defaultHeight)
	l.Title = "Select AWS Region"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = styles.TitleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	return RegionSelectorModel{
		list: l,
	}
}

// Init initializes the region selector
func (m RegionSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the region selector
func (m RegionSelectorModel) Update(msg tea.Msg) (RegionSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.list.SelectedItem()
			if selected != nil {
				item := selected.(regionItem)
				m.choice = item.region
				return m, func() tea.Msg {
					return types.RegionSelectedMsg{Region: item.region}
				}
			}
		case "backspace", "esc":
			return m, func() tea.Msg { return types.BackMsg{} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the region selector
func (m RegionSelectorModel) View() string {
	return m.list.View()
}

// SetSize updates the dimensions of the region selector
func (m *RegionSelectorModel) SetSize(width, height int) {
	m.list.SetWidth(width)
	m.list.SetHeight(height - 2)
}

// SetDefaultRegion sets the default selected region if it exists in the list
func (m *RegionSelectorModel) SetDefaultRegion(region string) {
	if region == "" {
		return
	}

	// Find the index of the region in the list
	items := m.list.Items()
	for i, item := range items {
		if regionItem, ok := item.(regionItem); ok {
			if regionItem.region == region {
				m.list.Select(i)
				return
			}
		}
	}
}
