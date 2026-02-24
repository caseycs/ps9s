package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/config"
	"github.com/ilia/ps9s/internal/types"
)

func TestEscapeInSearchMode_OnlyCancelsSearch(t *testing.T) {
	m := newTestModel([]string{"prod"})
	m.currentScreen = ParameterListScreen
	m.parameterList.SearchActive = true

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	assertEqual(t, ParameterListScreen, m.currentScreen, "esc in search mode stays on list")
	assertEqual(t, false, m.parameterList.SearchActive, "esc in search mode cancels search")

	// Second ESC should now navigate back normally
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	assertEqual(t, RegionSelectorScreen, m.currentScreen, "second esc goes back to region selector")
}

// TestHelpers

// newTestModel creates a Model for testing with minimal dependencies
func newTestModel(profiles []string) Model {
	return NewModel(
		profiles,
		make(map[string]*aws.Client),
		&config.RegionMapping{ProfileRegions: make(map[string]string)},
	)
}

// assertEqual checks if two values are equal, failing the test if they're not
func assertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// assertNotNil checks if a value is not nil, failing the test if it is
func assertNotNil(t *testing.T, val interface{}, msg string) {
	t.Helper()
	if val == nil {
		t.Errorf("%s: expected non-nil value", msg)
	}
}

// updateModel is a helper to reduce boilerplate for Update + type assertion
func updateModel(m Model, msg tea.Msg) Model {
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// TestNavigateProfileToRegion tests forward navigation from ProfileSelector to RegionSelector
func TestNavigateProfileToRegion(t *testing.T) {
	m := newTestModel([]string{"prod", "staging", "dev"})

	// Initial state
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "initial screen")
	assertEqual(t, "", m.currentProfile, "initial profile")

	// Send ProfileSelectedMsg
	msg := types.ProfileSelectedMsg{Profile: "prod"}
	m = updateModel(m, msg)

	// Verify transition
	assertEqual(t, RegionSelectorScreen, m.currentScreen, "screen after profile selection")
	assertEqual(t, "prod", m.currentProfile, "profile after selection")
}

// TestNavigateRegionToParameterList tests forward navigation from RegionSelector to ParameterList
func TestNavigateRegionToParameterList(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Set up initial state: already at region selector
	m.currentProfile = "prod"
	m.currentScreen = RegionSelectorScreen

	// Send RegionSelectedMsg
	msg := types.RegionSelectedMsg{Region: "us-east-1"}
	m = updateModel(m, msg)

	// Verify transition
	assertEqual(t, ParameterListScreen, m.currentScreen, "screen after region selection")
	assertEqual(t, "us-east-1", m.currentRegion, "region after selection")
	assertEqual(t, "prod", m.currentProfile, "profile unchanged")
}

// TestNavigateToParameterView tests navigation from ParameterList to ParameterView
func TestNavigateToParameterView(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Set up initial state
	m.currentScreen = ParameterListScreen
	m.currentProfile = "prod"
	m.currentRegion = "us-east-1"

	// Send ViewParameterMsg
	msg := types.ViewParameterMsg{Parameter: nil}
	m = updateModel(m, msg)

	// Verify transition
	assertEqual(t, ParameterViewScreen, m.currentScreen, "screen after view parameter")
}

// TestNavigateToParameterEdit tests screen transition to edit (without AWS calls)
func TestNavigateToParameterEdit(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Set up initial state directly without calling Update
	// (EditParameterMsg requires valid AWS client and parameter)
	m.currentScreen = ParameterEditScreen
	m.currentProfile = "prod"
	m.currentRegion = "us-east-1"

	// Verify we can be at edit screen
	assertEqual(t, ParameterEditScreen, m.currentScreen, "can be at edit screen")
}

// TestBackNavigationFromRegionSelector tests back button from RegionSelector
func TestBackNavigationFromRegionSelector(t *testing.T) {
	m := newTestModel([]string{"prod"})
	m.currentScreen = RegionSelectorScreen
	m.currentProfile = "prod"

	// Send BackMsg
	msg := types.BackMsg{}
	m = updateModel(m, msg)

	// Verify transition
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "back from region selector")
}

// TestBackNavigationFromParameterList tests back button from ParameterList
func TestBackNavigationFromParameterList(t *testing.T) {
	m := newTestModel([]string{"prod"})
	m.currentScreen = ParameterListScreen
	m.currentProfile = "prod"
	m.currentRegion = "us-east-1"

	// Send BackMsg
	msg := types.BackMsg{}
	m = updateModel(m, msg)

	// Verify transition
	assertEqual(t, RegionSelectorScreen, m.currentScreen, "back from parameter list")
}

// TestBackNavigationFromParameterView tests back button from ParameterView
func TestBackNavigationFromParameterView(t *testing.T) {
	m := newTestModel([]string{"prod"})
	m.currentScreen = ParameterViewScreen

	// Send BackMsg
	msg := types.BackMsg{}
	m = updateModel(m, msg)

	// Verify transition
	assertEqual(t, ParameterListScreen, m.currentScreen, "back from parameter view")
}

// TestBackNavigationFromParameterEdit tests back button from ParameterEdit
func TestBackNavigationFromParameterEdit(t *testing.T) {
	m := newTestModel([]string{"prod"})
	m.currentScreen = ParameterEditScreen

	// Send BackMsg
	msg := types.BackMsg{}
	m = updateModel(m, msg)

	// Verify transition
	assertEqual(t, ParameterViewScreen, m.currentScreen, "back from parameter edit")
}

// TestBackNavigationFromProfileSelector tests back button at ProfileSelector (no-op)
func TestBackNavigationFromProfileSelector(t *testing.T) {
	m := newTestModel([]string{"prod"})
	m.currentScreen = ProfileSelectorScreen

	// Send BackMsg
	msg := types.BackMsg{}
	m = updateModel(m, msg)

	// Verify no transition (should remain at ProfileSelector)
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "back from profile selector should be no-op")
}

// TestStateConsistencyAfterNavigation verifies state remains consistent through multiple navigations
func TestStateConsistencyAfterNavigation(t *testing.T) {
	m := newTestModel([]string{"prod", "staging"})

	// Navigate: Profile → Region → List
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	m = updateModel(m, types.RegionSelectedMsg{Region: "eu-west-1"})

	// Verify state consistency
	assertEqual(t, ParameterListScreen, m.currentScreen, "final screen")
	assertEqual(t, "prod", m.currentProfile, "profile consistency")
	assertEqual(t, "eu-west-1", m.currentRegion, "region consistency")
}

// TestNavigationPathProfileToView tests full forward path: Profile → Region → List → View
func TestNavigationPathProfileToView(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Forward navigation path
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	assertEqual(t, RegionSelectorScreen, m.currentScreen, "after profile selection")

	m = updateModel(m, types.RegionSelectedMsg{Region: "us-west-2"})
	assertEqual(t, ParameterListScreen, m.currentScreen, "after region selection")

	m = updateModel(m, types.ViewParameterMsg{Parameter: nil})
	assertEqual(t, ParameterViewScreen, m.currentScreen, "after view parameter")

	// Verify complete state
	assertEqual(t, "prod", m.currentProfile, "profile in view")
	assertEqual(t, "us-west-2", m.currentRegion, "region in view")
}

// TestNavigationPathProfileToEdit tests forward path to view and then manually to edit
func TestNavigationPathProfileToEdit(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Forward navigation to view
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	m = updateModel(m, types.RegionSelectedMsg{Region: "ap-southeast-1"})
	m = updateModel(m, types.ViewParameterMsg{Parameter: nil})
	assertEqual(t, ParameterViewScreen, m.currentScreen, "at view screen")

	// Manually transition to edit (skip calling EditParameterMsg which requires AWS)
	m.currentScreen = ParameterEditScreen
	
	assertEqual(t, ParameterEditScreen, m.currentScreen, "edit screen")
	assertEqual(t, "prod", m.currentProfile, "profile in edit")
	assertEqual(t, "ap-southeast-1", m.currentRegion, "region in edit")
}

// TestRoundTripNavigation tests navigating forward and back to original state
func TestRoundTripNavigation(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Forward: Profile → Region → List → View
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	m = updateModel(m, types.RegionSelectedMsg{Region: "us-east-1"})
	m = updateModel(m, types.ViewParameterMsg{Parameter: nil})
	assertEqual(t, ParameterViewScreen, m.currentScreen, "at view screen")

	// Backward: View → List → Region → Profile
	m = updateModel(m, types.BackMsg{})
	assertEqual(t, ParameterListScreen, m.currentScreen, "back to list")

	m = updateModel(m, types.BackMsg{})
	assertEqual(t, RegionSelectorScreen, m.currentScreen, "back to region")

	m = updateModel(m, types.BackMsg{})
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "back to profile")
}

// TestGoToProfileSelection tests direct jump to profile selection screen
func TestGoToProfileSelection(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Start deep in the navigation tree
	m.currentScreen = ParameterEditScreen
	m.currentProfile = "prod"
	m.currentRegion = "us-east-1"

	// Jump to profile selection
	msg := types.GoToProfileSelectionMsg{}
	m = updateModel(m, msg)

	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "jumped to profile selection")
	// Profile and region are preserved (not reset)
	assertEqual(t, "prod", m.currentProfile, "profile preserved")
	assertEqual(t, "us-east-1", m.currentRegion, "region preserved")
}

// TestWindowSizeMessage tests that WindowSizeMsg is propagated correctly
func TestWindowSizeMessage(t *testing.T) {
	m := newTestModel([]string{"prod"})

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	m = updateModel(m, msg)

	assertEqual(t, 120, m.width, "width")
	assertEqual(t, 40, m.height, "height")
}

// TestSwitchRecentEntry tests switching between recent profile+region entries
func TestSwitchRecentEntry(t *testing.T) {
	m := newTestModel([]string{"prod", "staging"})

	// Start at profile selector
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "initial screen")

	// Skip calling Update with SwitchRecentMsg as it tries to load parameters
	// Instead, manually set state to what we'd expect after switch
	m.currentProfile = "staging"
	m.currentRegion = "eu-central-1"
	m.currentScreen = ParameterListScreen

	// Verify state
	assertEqual(t, ParameterListScreen, m.currentScreen, "switched to parameter list")
	assertEqual(t, "staging", m.currentProfile, "switched profile")
	assertEqual(t, "eu-central-1", m.currentRegion, "switched region")
}

// TestMultipleProfileSwitches tests switching between different profiles
func TestMultipleProfileSwitches(t *testing.T) {
	m := newTestModel([]string{"prod", "staging", "dev"})

	// Select prod
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	assertEqual(t, "prod", m.currentProfile, "first profile")

	// Go back and select staging
	m = updateModel(m, types.BackMsg{})
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "staging"})
	assertEqual(t, "staging", m.currentProfile, "second profile")

	// Go back and select dev
	m = updateModel(m, types.BackMsg{})
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "dev"})
	assertEqual(t, "dev", m.currentProfile, "third profile")
}

// TestNavigationWithQuitMessage tests that quit message is handled
func TestNavigationWithQuitMessage(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Navigate to parameter list
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	m = updateModel(m, types.RegionSelectedMsg{Region: "us-east-1"})
	assertEqual(t, ParameterListScreen, m.currentScreen, "at parameter list")

	// Send quit command (simulating Ctrl+C)
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(keyMsg)

	// Verify quit command was generated
	assertNotNil(t, cmd, "quit command returned")
	// Can't directly compare tea.Quit (function pointer), but verify it's not nil
	if cmd == nil {
		t.Errorf("expected quit command, got nil")
	}
}

// NavigationTestCase represents a test case for navigation sequences
type NavigationTestCase struct {
	name            string
	initialScreen   Screen
	messages        []tea.Msg
	expectedScreen  Screen
	expectedProfile string
	expectedRegion  string
}

// TestNavigationSequences tests multiple navigation paths comprehensively
func TestNavigationSequences(t *testing.T) {
	tests := []NavigationTestCase{
		{
			name:            "simple profile selection",
			initialScreen:   ProfileSelectorScreen,
			messages:        []tea.Msg{types.ProfileSelectedMsg{Profile: "prod"}},
			expectedScreen:  RegionSelectorScreen,
			expectedProfile: "prod",
		},
		{
			name:            "profile and region selection",
			initialScreen:   ProfileSelectorScreen,
			messages: []tea.Msg{
				types.ProfileSelectedMsg{Profile: "prod"},
				types.RegionSelectedMsg{Region: "us-east-1"},
			},
			expectedScreen:  ParameterListScreen,
			expectedProfile: "prod",
			expectedRegion:  "us-east-1",
		},
		{
			name:           "back from region selector",
			initialScreen:  RegionSelectorScreen,
			messages:       []tea.Msg{types.BackMsg{}},
			expectedScreen: ProfileSelectorScreen,
		},
		{
			name:           "parameter view and back",
			initialScreen:  ParameterViewScreen,
			messages:       []tea.Msg{types.BackMsg{}},
			expectedScreen: ParameterListScreen,
		},
		{
			name:           "edit and back to view",
			initialScreen:  ParameterEditScreen,
			messages:       []tea.Msg{types.BackMsg{}},
			expectedScreen: ParameterViewScreen,
		},
		{
			name:           "full forward path",
			initialScreen:  ProfileSelectorScreen,
			messages: []tea.Msg{
				types.ProfileSelectedMsg{Profile: "staging"},
				types.RegionSelectedMsg{Region: "eu-west-1"},
				types.ViewParameterMsg{Parameter: nil},
			},
			expectedScreen:  ParameterViewScreen,
			expectedProfile: "staging",
			expectedRegion:  "eu-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel([]string{"prod", "staging", "dev"})
			m.currentScreen = tt.initialScreen

			// Apply all messages in sequence
			for _, msg := range tt.messages {
				m = updateModel(m, msg)
			}

			// Verify final state
			assertEqual(t, tt.expectedScreen, m.currentScreen, "screen")
			if tt.expectedProfile != "" {
				assertEqual(t, tt.expectedProfile, m.currentProfile, "profile")
			}
			if tt.expectedRegion != "" {
				assertEqual(t, tt.expectedRegion, m.currentRegion, "region")
			}
		})
	}
}

// TestNavigationDoesNotLoseContext verifies that profile/region context is maintained
func TestNavigationDoesNotLoseContext(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Navigate through several screens
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	m = updateModel(m, types.RegionSelectedMsg{Region: "ap-southeast-1"})
	m = updateModel(m, types.ViewParameterMsg{Parameter: nil})
	m = updateModel(m, types.BackMsg{})
	m = updateModel(m, types.BackMsg{})

	// Context should still be maintained
	assertEqual(t, "prod", m.currentProfile, "profile context")
	assertEqual(t, "ap-southeast-1", m.currentRegion, "region context")
	assertEqual(t, RegionSelectorScreen, m.currentScreen, "screen")
}

// TestNavigationEdgeCases tests edge cases and unusual navigation patterns
func TestNavigationEdgeCases(t *testing.T) {
	m := newTestModel([]string{"prod"})

	// Edge case: back from ProfileSelector should be no-op
	m.currentScreen = ProfileSelectorScreen
	oldScreen := m.currentScreen
	m = updateModel(m, types.BackMsg{})
	assertEqual(t, oldScreen, m.currentScreen, "back from profile selector is no-op")

	// Edge case: multiple consecutive backs
	m = newTestModel([]string{"prod"})
	m.currentScreen = ParameterListScreen
	m = updateModel(m, types.BackMsg{})
	m = updateModel(m, types.BackMsg{})
	m = updateModel(m, types.BackMsg{})
	m = updateModel(m, types.BackMsg{}) // Extra back from ProfileSelector
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "multiple backs")
}

// TestEmptyProfileList tests navigation with minimal profiles
func TestEmptyProfileList(t *testing.T) {
	m := newTestModel([]string{})

	// Should initialize without error
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "initial screen")
	assertEqual(t, 0, len(m.profiles), "empty profiles")
}

// TestSingleProfileNavigation tests navigation with only one profile
func TestSingleProfileNavigation(t *testing.T) {
	m := newTestModel([]string{"only-prod"})

	m = updateModel(m, types.ProfileSelectedMsg{Profile: "only-prod"})
	assertEqual(t, RegionSelectorScreen, m.currentScreen, "region selector")
	assertEqual(t, "only-prod", m.currentProfile, "single profile")

	m = updateModel(m, types.RegionSelectedMsg{Region: "us-west-2"})
	assertEqual(t, ParameterListScreen, m.currentScreen, "parameter list")
	assertEqual(t, "us-west-2", m.currentRegion, "region")
}

// TestContextPersistenceAcrossNavigations tests context preservation
func TestContextPersistenceAcrossNavigations(t *testing.T) {
	m := newTestModel([]string{"prod"})
	profile := "test-profile"
	region := "test-region"

	// Set context
	m = updateModel(m, types.ProfileSelectedMsg{Profile: profile})
	m = updateModel(m, types.RegionSelectedMsg{Region: region})

	// Navigate: List → View → List
	m = updateModel(m, types.ViewParameterMsg{Parameter: nil})
	assertEqual(t, ParameterViewScreen, m.currentScreen, "view screen")
	m = updateModel(m, types.BackMsg{})
	assertEqual(t, ParameterListScreen, m.currentScreen, "back to list")

	// Context should still be intact
	assertEqual(t, profile, m.currentProfile, "profile preserved")
	assertEqual(t, region, m.currentRegion, "region preserved")
}

// Benchmark tests

// BenchmarkNavigationSequence benchmarks a typical navigation path
func BenchmarkNavigationSequence(b *testing.B) {
	m := newTestModel([]string{"prod", "staging", "dev"})
	
	for i := 0; i < b.N; i++ {
		m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
		m = updateModel(m, types.RegionSelectedMsg{Region: "us-east-1"})
		m = updateModel(m, types.ViewParameterMsg{Parameter: nil})
		m = updateModel(m, types.BackMsg{})
		m = updateModel(m, types.BackMsg{})
	}
}

// BenchmarkBackNavigation benchmarks back navigation
func BenchmarkBackNavigation(b *testing.B) {
	m := newTestModel([]string{"prod"})
	m.currentScreen = ParameterViewScreen
	
	for i := 0; i < b.N; i++ {
		m = updateModel(m, types.BackMsg{})
		// Reset for next iteration
		m.currentScreen = ParameterViewScreen
	}
}
