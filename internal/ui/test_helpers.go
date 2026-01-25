package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/config"
	"github.com/ilia/ps9s/internal/types"
)

// TestModelBuilder provides a fluent interface for constructing test models with specific state
type TestModelBuilder struct {
	profiles   []string
	clients    map[string]*aws.Client
	regions    *config.RegionMapping
	screen     Screen
	profile    string
	region     string
	width      int
	height     int
}

// NewTestModelBuilder creates a new builder for constructing test models
func NewTestModelBuilder() *TestModelBuilder {
	return &TestModelBuilder{
		profiles: []string{"prod", "staging", "dev"},
		clients:  make(map[string]*aws.Client),
		regions:  &config.RegionMapping{ProfileRegions: make(map[string]string)},
		screen:   ProfileSelectorScreen,
		width:    80,
		height:   24,
	}
}

// WithProfiles sets the profiles list
func (b *TestModelBuilder) WithProfiles(profiles ...string) *TestModelBuilder {
	b.profiles = profiles
	return b
}

// WithScreen sets the current screen
func (b *TestModelBuilder) WithScreen(screen Screen) *TestModelBuilder {
	b.screen = screen
	return b
}

// WithProfile sets the current profile
func (b *TestModelBuilder) WithProfile(profile string) *TestModelBuilder {
	b.profile = profile
	return b
}

// WithRegion sets the current region
func (b *TestModelBuilder) WithRegion(region string) *TestModelBuilder {
	b.region = region
	return b
}

// WithDimensions sets the terminal dimensions
func (b *TestModelBuilder) WithDimensions(width, height int) *TestModelBuilder {
	b.width = width
	b.height = height
	return b
}

// Build constructs the Model with the configured state
func (b *TestModelBuilder) Build() Model {
	m := NewModel(b.profiles, b.clients, b.regions)
	m.currentScreen = b.screen
	m.currentProfile = b.profile
	m.currentRegion = b.region
	m.width = b.width
	m.height = b.height
	return m
}

// NavigationPath represents a sequence of messages and expected state transitions
type NavigationPath struct {
	name       string
	startState ModelState
	messages   []tea.Msg
	endState   ModelState
}

// ModelState captures the state of a Model at a point in time
type ModelState struct {
	Screen  Screen
	Profile string
	Region  string
}

// CaptureState returns the current state of a Model
func CaptureState(m Model) ModelState {
	return ModelState{
		Screen:  m.currentScreen,
		Profile: m.currentProfile,
		Region:  m.currentRegion,
	}
}

// AssertState checks if the model matches expected state
func AssertState(t *testing.T, m Model, expected ModelState, msg string) {
	t.Helper()
	actual := CaptureState(m)
	if actual != expected {
		t.Errorf("%s: expected %+v, got %+v", msg, expected, actual)
	}
}

// ExecuteNavigationPath applies all messages in sequence and verifies state transitions
func ExecuteNavigationPath(t *testing.T, m Model, path NavigationPath) Model {
	t.Helper()
	
	// Verify starting state
	AssertState(t, m, path.startState, path.name+": starting state")
	
	// Apply each message
	for i, msg := range path.messages {
		updated, _ := m.Update(msg)
		m = updated.(Model)
		t.Logf("%s: message %d applied: %T", path.name, i, msg)
	}
	
	// Verify ending state
	AssertState(t, m, path.endState, path.name+": ending state")
	
	return m
}

// CommonNavigationPaths provides pre-built navigation paths for testing
type CommonNavigationPaths struct{}

// ProfileToRegion returns a navigation path from profile selection to region selection
func (c CommonNavigationPaths) ProfileToRegion(profile string) NavigationPath {
	return NavigationPath{
		name: "profile to region",
		startState: ModelState{
			Screen: ProfileSelectorScreen,
		},
		messages: []tea.Msg{
			types.ProfileSelectedMsg{Profile: profile},
		},
		endState: ModelState{
			Screen:  RegionSelectorScreen,
			Profile: profile,
		},
	}
}

// RegionToParameterList returns a navigation path from region to parameter list
func (c CommonNavigationPaths) RegionToParameterList(profile, region string) NavigationPath {
	return NavigationPath{
		name: "region to parameter list",
		startState: ModelState{
			Screen:  RegionSelectorScreen,
			Profile: profile,
		},
		messages: []tea.Msg{
			types.RegionSelectedMsg{Region: region},
		},
		endState: ModelState{
			Screen:  ParameterListScreen,
			Profile: profile,
			Region:  region,
		},
	}
}

// BackNavigation returns a navigation path for going back
func (c CommonNavigationPaths) BackNavigation(from, to Screen) NavigationPath {
	return NavigationPath{
		name: "back navigation",
		startState: ModelState{
			Screen: from,
		},
		messages: []tea.Msg{
			types.BackMsg{},
		},
		endState: ModelState{
			Screen: to,
		},
	}
}

// FullPath returns a complete navigation path through multiple screens
func (c CommonNavigationPaths) FullPath(profile, region string) NavigationPath {
	return NavigationPath{
		name: "full navigation path",
		startState: ModelState{
			Screen: ProfileSelectorScreen,
		},
		messages: []tea.Msg{
			types.ProfileSelectedMsg{Profile: profile},
			types.RegionSelectedMsg{Region: region},
		},
		endState: ModelState{
			Screen:  ParameterListScreen,
			Profile: profile,
			Region:  region,
		},
	}
}

// RoundTrip returns a navigation path that goes forward and back
func (c CommonNavigationPaths) RoundTrip(profile, region string) NavigationPath {
	return NavigationPath{
		name: "round trip navigation",
		startState: ModelState{
			Screen: ProfileSelectorScreen,
		},
		messages: []tea.Msg{
			types.ProfileSelectedMsg{Profile: profile},
			types.RegionSelectedMsg{Region: region},
			types.BackMsg{},
			types.BackMsg{},
		},
		endState: ModelState{
			Screen:  ProfileSelectorScreen,
			Profile: profile,
			Region:  region,
		},
	}
}

// NavigationValidator provides utilities for validating navigation behavior
type NavigationValidator struct {
	t *testing.T
	m Model
}

// NewNavigationValidator creates a new validator for a model
func NewNavigationValidator(t *testing.T, m Model) *NavigationValidator {
	return &NavigationValidator{t: t, m: m}
}

// SendMessage applies a message and returns a new validator with the updated model
func (v *NavigationValidator) SendMessage(msg tea.Msg) *NavigationValidator {
	updated, _ := v.m.Update(msg)
	v.m = updated.(Model)
	return v
}

// AtScreen asserts that the model is at a specific screen
func (v *NavigationValidator) AtScreen(screen Screen) *NavigationValidator {
	if v.m.currentScreen != screen {
		v.t.Errorf("expected screen %d, got %d", screen, v.m.currentScreen)
	}
	return v
}

// WithProfile asserts that the model has a specific profile
func (v *NavigationValidator) WithProfile(profile string) *NavigationValidator {
	if v.m.currentProfile != profile {
		v.t.Errorf("expected profile %q, got %q", profile, v.m.currentProfile)
	}
	return v
}

// WithRegion asserts that the model has a specific region
func (v *NavigationValidator) WithRegion(region string) *NavigationValidator {
	if v.m.currentRegion != region {
		v.t.Errorf("expected region %q, got %q", region, v.m.currentRegion)
	}
	return v
}

// GetModel returns the underlying model (for advanced testing)
func (v *NavigationValidator) GetModel() Model {
	return v.m
}

// ScreenName returns a human-readable name for a screen
func ScreenName(s Screen) string {
	return screenName(s)
}
