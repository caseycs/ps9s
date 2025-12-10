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

// profileItem represents a profile in the list
type profileItem struct {
	profile string
}

func (i profileItem) FilterValue() string { return i.profile }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(profileItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.profile)

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

// ProfileSelectorModel represents the profile selection screen
type ProfileSelectorModel struct {
	list   list.Model
	choice string
}

// NewProfileSelector creates a new profile selector screen
func NewProfileSelector(profiles []string) ProfileSelectorModel {
	items := make([]list.Item, len(profiles))
	for i, p := range profiles {
		items[i] = profileItem{profile: p}
	}

	const defaultWidth = 80
	const defaultHeight = 20

	l := list.New(items, itemDelegate{}, defaultWidth, defaultHeight)
	l.Title = "Select AWS Profile"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = styles.TitleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	return ProfileSelectorModel{
		list: l,
	}
}

// Init initializes the profile selector
func (m ProfileSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the profile selector
func (m ProfileSelectorModel) Update(msg tea.Msg) (ProfileSelectorModel, tea.Cmd) {
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
				item := selected.(profileItem)
				m.choice = item.profile
				return m, func() tea.Msg {
					return types.ProfileSelectedMsg{Profile: item.profile}
				}
			}
		case "esc", "backspace":
			// Don't quit on escape - user must use 'q' to quit
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the profile selector
func (m ProfileSelectorModel) View() string {
	return m.list.View()
}

// SetSize updates the dimensions of the profile selector
func (m *ProfileSelectorModel) SetSize(width, height int) {
	m.list.SetWidth(width)
	m.list.SetHeight(height - 2)
}
