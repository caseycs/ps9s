package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilia/ps9s/internal/types"
)

// TestNavigationWithBuilder tests using the fluent TestModelBuilder
func TestNavigationWithBuilder(t *testing.T) {
	m := NewTestModelBuilder().
		WithProfiles("prod", "staging", "dev").
		WithScreen(ProfileSelectorScreen).
		WithDimensions(120, 40).
		Build()

	assertEqual(t, 120, m.width, "builder dimensions")
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "builder initial screen")
}

// TestNavigationWithValidator tests using the fluent NavigationValidator
func TestNavigationWithValidator(t *testing.T) {
	m := NewTestModelBuilder().
		WithProfiles("prod", "staging").
		Build()

	NewNavigationValidator(t, m).
		SendMessage(types.ProfileSelectedMsg{Profile: "prod"}).
		AtScreen(RegionSelectorScreen).
		WithProfile("prod").
		SendMessage(types.RegionSelectedMsg{Region: "us-east-1"}).
		AtScreen(ParameterListScreen).
		WithRegion("us-east-1").
		SendMessage(types.BackMsg{}).
		AtScreen(RegionSelectorScreen)
}

// TestComplexNavigationSequence tests a complex multi-step navigation
func TestComplexNavigationSequence(t *testing.T) {
	m := NewTestModelBuilder().
		WithProfiles("prod", "staging", "dev").
		Build()

	paths := CommonNavigationPaths{}

	// Test profile to region transition
	m = ExecuteNavigationPath(t, m, paths.ProfileToRegion("prod"))

	// Reset and test full path
	m = NewTestModelBuilder().WithProfiles("prod", "staging", "dev").Build()
	m = ExecuteNavigationPath(t, m, paths.FullPath("staging", "eu-west-1"))
}

// TestMultipleProfileNavigation tests switching between multiple profiles with state preservation
func TestMultipleProfileNavigation(t *testing.T) {
	m := NewTestModelBuilder().WithProfiles("prod", "staging", "dev").Build()

	validator := NewNavigationValidator(t, m)

	// Navigate through prod
	validator.
		SendMessage(types.ProfileSelectedMsg{Profile: "prod"}).
		AtScreen(RegionSelectorScreen).
		WithProfile("prod")

	// Reset to profile selector
	validator.SendMessage(types.BackMsg{}).AtScreen(ProfileSelectorScreen)

	// Switch to staging
	validator.
		SendMessage(types.ProfileSelectedMsg{Profile: "staging"}).
		AtScreen(RegionSelectorScreen).
		WithProfile("staging")

	// Reset again
	validator.SendMessage(types.BackMsg{}).AtScreen(ProfileSelectorScreen)

	// Switch to dev
	validator.
		SendMessage(types.ProfileSelectedMsg{Profile: "dev"}).
		AtScreen(RegionSelectorScreen).
		WithProfile("dev")
}

// TestContextPreservation tests that profile and region context is preserved across navigation
func TestContextPreservation(t *testing.T) {
	m := NewTestModelBuilder().
		WithProfiles("prod", "staging").
		Build()

	validator := NewNavigationValidator(t, m)

	// Navigate to parameter list
	validator.
		SendMessage(types.ProfileSelectedMsg{Profile: "prod"}).
		SendMessage(types.RegionSelectedMsg{Region: "us-west-2"}).
		AtScreen(ParameterListScreen).
		WithProfile("prod").
		WithRegion("us-west-2")

	// Navigate to view
	validator.
		SendMessage(types.ViewParameterMsg{Parameter: nil}).
		AtScreen(ParameterViewScreen).
		WithProfile("prod").
		WithRegion("us-west-2")

	// Navigate back to list - context should be preserved
	validator.
		SendMessage(types.BackMsg{}).
		AtScreen(ParameterListScreen).
		WithProfile("prod").
		WithRegion("us-west-2")
}

// TestNavigationBoundaries tests navigation at screen boundaries
func TestNavigationBoundaries(t *testing.T) {
	m := NewTestModelBuilder().Build()

	validator := NewNavigationValidator(t, m)

	// At ProfileSelector, pressing back should be no-op
	beforeState := CaptureState(validator.GetModel())
	validator.SendMessage(types.BackMsg{})
	afterState := CaptureState(validator.GetModel())

	if beforeState != afterState {
		t.Errorf("back from ProfileSelector should be no-op: before %+v, after %+v", beforeState, afterState)
	}
}

// TestWindowSizeHandling tests that window size messages are handled correctly
func TestWindowSizeHandling(t *testing.T) {
	m := NewTestModelBuilder().
		WithDimensions(80, 24).
		Build()

	validator := NewNavigationValidator(t, m)
	model := validator.GetModel()

	assertEqual(t, 80, model.width, "initial width")
	assertEqual(t, 24, model.height, "initial height")

	// Send window size message
	validator.SendMessage(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = validator.GetModel()

	assertEqual(t, 120, model.width, "updated width")
	assertEqual(t, 40, model.height, "updated height")
}

// TestQuickNavigationSequences tests various rapid navigation sequences
func TestQuickNavigationSequences(t *testing.T) {
	sequences := []struct {
		name     string
		messages []tea.Msg
		profile  string
		region   string
	}{
		{
			name: "profile then region",
			messages: []tea.Msg{
				types.ProfileSelectedMsg{Profile: "prod"},
				types.RegionSelectedMsg{Region: "us-east-1"},
			},
			profile: "prod",
			region:  "us-east-1",
		},
		{
			name: "different profile and region",
			messages: []tea.Msg{
				types.ProfileSelectedMsg{Profile: "staging"},
				types.RegionSelectedMsg{Region: "eu-west-1"},
			},
			profile: "staging",
			region:  "eu-west-1",
		},
		{
			name: "multiple profiles with back",
			messages: []tea.Msg{
				types.ProfileSelectedMsg{Profile: "prod"},
				types.BackMsg{},
				types.ProfileSelectedMsg{Profile: "dev"},
				types.RegionSelectedMsg{Region: "ap-southeast-1"},
			},
			profile: "dev",
			region:  "ap-southeast-1",
		},
	}

	for _, seq := range sequences {
		t.Run(seq.name, func(t *testing.T) {
			m := NewTestModelBuilder().
				WithProfiles("prod", "staging", "dev").
				Build()

			for _, msg := range seq.messages {
				updated, _ := m.Update(msg)
				m = updated.(Model)
			}

			assertEqual(t, seq.profile, m.currentProfile, "profile")
			assertEqual(t, seq.region, m.currentRegion, "region")
		})
	}
}

// TestStatefulNavigation tests navigation with state preservation across complex paths
func TestStatefulNavigation(t *testing.T) {
	m := NewTestModelBuilder().
		WithProfiles("prod", "staging", "dev").
		Build()

	// Path 1: prod -> us-east-1 -> view -> back to list
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
	m = updateModel(m, types.RegionSelectedMsg{Region: "us-east-1"})
	m = updateModel(m, types.ViewParameterMsg{Parameter: nil})

	state1 := CaptureState(m)
	assertEqual(t, ParameterViewScreen, state1.Screen, "at view screen")
	assertEqual(t, "prod", state1.Profile, "profile in view")
	assertEqual(t, "us-east-1", state1.Region, "region in view")

	// Go back to list
	m = updateModel(m, types.BackMsg{})
	state2 := CaptureState(m)
	assertEqual(t, ParameterListScreen, state2.Screen, "back to list")
	assertEqual(t, "prod", state2.Profile, "profile preserved")
	assertEqual(t, "us-east-1", state2.Region, "region preserved")

	// Go back to region selector
	m = updateModel(m, types.BackMsg{})
	state3 := CaptureState(m)
	assertEqual(t, RegionSelectorScreen, state3.Screen, "back to region selector")
	assertEqual(t, "prod", state3.Profile, "profile still preserved")

	// Go back to profile selector
	m = updateModel(m, types.BackMsg{})
	state4 := CaptureState(m)
	assertEqual(t, ProfileSelectorScreen, state4.Screen, "back to profile selector")
	assertEqual(t, "prod", state4.Profile, "profile retained")
	assertEqual(t, "us-east-1", state4.Region, "region retained")

	// Switch to different profile and navigate
	m = updateModel(m, types.ProfileSelectedMsg{Profile: "staging"})
	state5 := CaptureState(m)
	assertEqual(t, RegionSelectorScreen, state5.Screen, "switched to staging at region selector")
	assertEqual(t, "staging", state5.Profile, "new profile set")
	assertEqual(t, "us-east-1", state5.Region, "old region preserved (not cleared)")
}

// TestGoToProfileSelectionFromDeepState tests jumping directly to profile selection
func TestGoToProfileSelectionFromDeepState(t *testing.T) {
	m := NewTestModelBuilder().
		WithProfiles("prod", "staging").
		WithScreen(ParameterViewScreen).
		WithProfile("prod").
		WithRegion("us-east-1").
		Build()

	// Start at deep state
	assertEqual(t, ParameterViewScreen, m.currentScreen, "starting at view screen")

	// Jump to profile selection
	m = updateModel(m, types.GoToProfileSelectionMsg{})

	// Should be at profile selector
	assertEqual(t, ProfileSelectorScreen, m.currentScreen, "jumped to profile selector")
	// But profile and region should still be preserved for later use
	assertEqual(t, "prod", m.currentProfile, "profile preserved after jump")
	assertEqual(t, "us-east-1", m.currentRegion, "region preserved after jump")
}

// BenchmarkComplexNavigation benchmarks a complex navigation sequence
func BenchmarkComplexNavigation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewTestModelBuilder().
			WithProfiles("prod", "staging", "dev").
			Build()

		// Simulate user navigating through multiple screens
		m = updateModel(m, types.ProfileSelectedMsg{Profile: "prod"})
		m = updateModel(m, types.RegionSelectedMsg{Region: "us-east-1"})
		m = updateModel(m, types.ViewParameterMsg{Parameter: nil})
		m = updateModel(m, types.BackMsg{})
		m = updateModel(m, types.BackMsg{})
		m = updateModel(m, types.ProfileSelectedMsg{Profile: "staging"})
		m = updateModel(m, types.RegionSelectedMsg{Region: "eu-west-1"})
		m = updateModel(m, types.BackMsg{})
		m = updateModel(m, types.BackMsg{})
	}
}

// BenchmarkValidatorChain benchmarks using the fluent validator interface
func BenchmarkValidatorChain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewTestModelBuilder().
			WithProfiles("prod", "staging", "dev").
			Build()

		NewNavigationValidator(&testing.T{}, m).
			SendMessage(types.ProfileSelectedMsg{Profile: "prod"}).
			SendMessage(types.RegionSelectedMsg{Region: "us-east-1"}).
			SendMessage(types.ViewParameterMsg{Parameter: nil}).
			SendMessage(types.BackMsg{}).
			SendMessage(types.BackMsg{}).
			SendMessage(types.ProfileSelectedMsg{Profile: "staging"}).
			SendMessage(types.RegionSelectedMsg{Region: "eu-west-1"})
	}
}
