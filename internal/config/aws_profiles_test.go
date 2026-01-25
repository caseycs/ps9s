package config

import (
	"strings"
	"testing"
)

func TestParseAWSConfigProfiles(t *testing.T) {
	config := `
[default]
region = us-east-1

[profile staging]
region = us-west-2

[sso-session mysession]
sso_start_url = https://example.com

[profile prod]
region = eu-central-1
`

	got := parseAWSConfigProfiles(strings.NewReader(config))
	want := []string{"default", "prod", "staging"}

	if len(got) != len(want) {
		t.Fatalf("expected %d profiles, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %q at index %d, got %q (full=%#v)", want[i], i, got[i], got)
		}
	}
}

func TestParseAWSConfigProfiles_DedupAndIgnoreJunk(t *testing.T) {
	config := `
# comment
[profile dev]

[profile dev] ; duplicate

[profile ]

not a section

[default]
`

	got := parseAWSConfigProfiles(strings.NewReader(config))
	want := []string{"default", "dev"}

	if len(got) != len(want) {
		t.Fatalf("expected %d profiles, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %q at index %d, got %q (full=%#v)", want[i], i, got[i], got)
		}
	}
}
