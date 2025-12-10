package types

import "github.com/ilia/ps9s/internal/aws"

// ProfileSelectedMsg is sent when a user selects an AWS profile
type ProfileSelectedMsg struct {
	Profile string
}

// RegionSelectedMsg is sent when a user selects an AWS region
type RegionSelectedMsg struct {
	Region string
}

// ViewParameterMsg is sent when a user wants to view a parameter
type ViewParameterMsg struct {
	Parameter *aws.Parameter
}

// EditParameterMsg is sent when a user wants to edit a parameter
type EditParameterMsg struct {
	Parameter *aws.Parameter
	JSONKey   string // Optional: if set, edit only this JSON key
}

// BackMsg is sent when a user wants to go back to the previous screen
type BackMsg struct{}

// SaveSuccessMsg is sent when a parameter is successfully saved
type SaveSuccessMsg struct {
	Parameter *aws.Parameter
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

// ParametersLoadedMsg is sent when parameters are loaded from AWS
type ParametersLoadedMsg struct {
	Parameters []*aws.Parameter
}

// ParameterValueLoadedMsg is sent when a parameter value is loaded
type ParameterValueLoadedMsg struct {
	Parameter *aws.Parameter
}

// SwitchRecentMsg is sent when user selects a recent profile+region entry
type SwitchRecentMsg struct {
	Profile string
	Region  string
}

// GoToProfileSelectionMsg is sent when user wants to jump to profile selection
type GoToProfileSelectionMsg struct{}
