package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/config"
	"github.com/ilia/ps9s/internal/types"
	"github.com/ilia/ps9s/internal/ui/screens"
)

// Screen represents the current screen being displayed
type Screen int

const (
	ProfileSelectorScreen Screen = iota
	RegionSelectorScreen
	ParameterListScreen
	ParameterViewScreen
	ParameterEditScreen
)

// Model represents the root application model
type Model struct {
	currentScreen Screen

	// Screen models
	profileSelector screens.ProfileSelectorModel
	regionSelector  screens.RegionSelectorModel
	parameterList   screens.ParameterListModel
	parameterView   screens.ParameterViewModel
	parameterEdit   screens.ParameterEditModel

	// Shared state
	profiles       []string
	currentProfile string
	currentRegion  string
	awsClients     map[string]*aws.Client
	regionMapping  *config.RegionMapping
	// Recent profile+region entries (most recent first)
	recents []config.RecentEntry
	// Flag to prevent reordering recents when switching via keyboard
	switchingToRecent bool

	// UI dimensions
	width, height int
}

// NewModel creates a new root model
func NewModel(profiles []string, clientPool map[string]*aws.Client, regionMapping *config.RegionMapping) Model {
	pl := screens.NewParameterList()

	// Load recents (non-fatal)
	recents, err := config.LoadRecentEntries()
	if err == nil {
		pl.SetRecents(recents)
	}

	return Model{
		currentScreen:   ProfileSelectorScreen,
		profileSelector: screens.NewProfileSelector(profiles),
		regionSelector:  screens.NewRegionSelector(),
		parameterList:   pl,
		parameterView:   screens.NewParameterView(),
		parameterEdit:   screens.NewParameterEdit(),
		profiles:        profiles,
		awsClients:      clientPool,
		regionMapping:   regionMapping,
		recents:         recents,
	}
}

// Init initializes the root model
func (m Model) Init() tea.Cmd {
	return m.profileSelector.Init()
}

// Update handles messages for the root model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Propagate size to all screens
		m.profileSelector.SetSize(msg.Width, msg.Height)
		m.regionSelector.SetSize(msg.Width, msg.Height)
		m.parameterList.SetSize(msg.Width, msg.Height)
		m.parameterView.SetSize(msg.Width, msg.Height)
		m.parameterEdit.SetSize(msg.Width, msg.Height)

	case types.ProfileSelectedMsg:
		m.currentProfile = msg.Profile
		m.currentScreen = RegionSelectorScreen
		// Set default region for this profile if it exists
		if lastRegion, ok := m.regionMapping.ProfileRegions[msg.Profile]; ok {
			m.regionSelector.SetDefaultRegion(lastRegion)
		}
		return m, nil

	case types.RegionSelectedMsg:
		m.currentRegion = msg.Region
		m.currentScreen = ParameterListScreen

		// Save the region selection for this profile
		m.regionMapping.ProfileRegions[m.currentProfile] = msg.Region
		if err := config.SaveRegionMapping(m.regionMapping); err != nil {
			// TODO: Show error in UI (non-fatal, continue anyway)
		}

		// Create/update client with selected region
		client, err := aws.NewClientWithRegion(context.Background(), m.currentProfile, msg.Region)
		if err != nil {
			// TODO: Show error in UI
			return m, nil
		}
		m.awsClients[m.currentProfile] = client

		// Pass profile/region context to parameter list screen
		m.parameterList.SetContext(m.currentProfile, msg.Region)

		return m, m.parameterList.LoadParameters(client)

	case types.ParametersLoadedMsg:
		// Only add to recents if we found parameters (don't add empty results)
		// and we're not switching to an existing recent entry (keep list stable)
		if len(msg.Parameters) > 0 && !m.switchingToRecent {
			entry := config.RecentEntry{Profile: m.currentProfile, Region: m.currentRegion}
			m.recents = config.AddRecentEntry(m.recents, entry, 5)
			_ = config.SaveRecentEntries(m.recents)
			m.parameterList.SetRecents(m.recents)
		}
		// Reset the flag after use
		m.switchingToRecent = false
		// Let the parameter list screen handle the actual parameter loading
		return m.updateCurrentScreen(msg)

	case types.ViewParameterMsg:
		m.currentScreen = ParameterViewScreen
		client := m.awsClients[m.currentProfile]
		// Pass profile/region context to parameter view
		m.parameterView.SetContext(m.currentProfile, m.currentRegion)
		return m, m.parameterView.LoadParameter(msg.Parameter, client)

	case types.EditParameterMsg:
		m.currentScreen = ParameterEditScreen
		client := m.awsClients[m.currentProfile]
		// Pass profile/region context tameter edit
		m.parameterEdit.SetContext(m.currentProfile, m.currentRegion)
		return m, m.parameterEdit.LoadParameter(msg.Parameter, client, msg.JSONKey)

	case types.SaveSuccessMsg:
		// Parameter saved successfully, update the view and go back
		// Ensure view has current profile/region
		m.parameterView.SetContext(m.currentProfile, m.currentRegion)
		// Load the updated parameter and return the command so Bubble Tea executes it
		cmd := m.parameterView.LoadParameter(msg.Parameter, m.awsClients[m.currentProfile])
		m.currentScreen = ParameterViewScreen
		return m, cmd

	case types.SwitchRecentMsg:
		// User selected a recent profile+region entry from the list
		m.currentProfile = msg.Profile
		m.currentRegion = msg.Region

		// Save region mapping
		m.regionMapping.ProfileRegions[m.currentProfile] = m.currentRegion
		_ = config.SaveRegionMapping(m.regionMapping)

		// Create/update client
		client, err := aws.NewClientWithRegion(context.Background(), m.currentProfile, m.currentRegion)
		if err != nil {
			// TODO: show error
			return m, nil
		}
		m.awsClients[m.currentProfile] = client

		// Don't reorder recents when switching via keyboard - keep list stable
		// The list only reorders when selecting from the profile/region screens
		m.switchingToRecent = true

		m.parameterList.SetContext(m.currentProfile, m.currentRegion)
		m.currentScreen = ParameterListScreen
		return m, m.parameterList.LoadParameters(client)

	case types.GoToProfileSelectionMsg:
		// Jump directly to profile selection screen
		m.currentScreen = ProfileSelectorScreen
		return m, nil

	case types.BackMsg:
		// Navigate back through screens
		switch m.currentScreen {
		case RegionSelectorScreen:
			m.currentScreen = ProfileSelectorScreen
		case ParameterListScreen:
			m.currentScreen = RegionSelectorScreen
		case ParameterViewScreen:
			m.currentScreen = ParameterListScreen
		case ParameterEditScreen:
			m.currentScreen = ParameterViewScreen
		}
		return m, nil

	case tea.KeyMsg:
		// Handle global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// Route to active screen
	return m.updateCurrentScreen(msg)
}

// updateCurrentScreen routes the message to the currently active screen
func (m Model) updateCurrentScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentScreen {
	case ProfileSelectorScreen:
		m.profileSelector, cmd = m.profileSelector.Update(msg)
	case RegionSelectorScreen:
		m.regionSelector, cmd = m.regionSelector.Update(msg)
	case ParameterListScreen:
		m.parameterList, cmd = m.parameterList.Update(msg)
	case ParameterViewScreen:
		m.parameterView, cmd = m.parameterView.Update(msg)
	case ParameterEditScreen:
		m.parameterEdit, cmd = m.parameterEdit.Update(msg)
	}

	return m, cmd
}

// View renders the current screen
func (m Model) View() string {
	switch m.currentScreen {
	case ProfileSelectorScreen:
		return m.profileSelector.View()
	case RegionSelectorScreen:
		return m.regionSelector.View()
	case ParameterListScreen:
		return m.parameterList.View()
	case ParameterViewScreen:
		return m.parameterView.View()
	case ParameterEditScreen:
		return m.parameterEdit.View()
	default:
		return "Unknown screen"
	}
}
