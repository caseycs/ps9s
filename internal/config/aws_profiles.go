package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GetProfilesFromAWSConfig returns AWS profile names from AWS_CONFIG_FILE or ~/.aws/config.
// If the config file can't be read or contains no profiles, it falls back to AWS_PROFILE
// (or "default") and returns a non-nil error describing the issue.
func GetProfilesFromAWSConfig() ([]string, error) {
	path, err := awsConfigPath()
	if err != nil {
		return fallbackProfiles(), err
	}

	f, err := os.Open(path)
	if err != nil {
		return fallbackProfiles(), fmt.Errorf("failed to open AWS config file %q: %w", path, err)
	}
	defer f.Close()

	profiles := parseAWSConfigProfiles(f)
	if len(profiles) == 0 {
		return fallbackProfiles(), fmt.Errorf("no AWS profiles found in %q", path)
	}

	return profiles, nil
}

func awsConfigPath() (string, error) {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".aws", "config"), nil
}

func fallbackProfiles() []string {
	if p := strings.TrimSpace(os.Getenv("AWS_PROFILE")); p != "" {
		return []string{p}
	}
	return []string{"default"}
}

func parseAWSConfigProfiles(r io.Reader) []string {
	seen := map[string]struct{}{}

	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}

		// Strip INI-style comments.
		if i := strings.IndexAny(line, "#;"); i >= 0 {
			line = strings.TrimSpace(line[:i])
			if line == "" {
				continue
			}
		}

		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}

		section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
		switch {
		case section == "default":
			seen["default"] = struct{}{}
		case strings.HasPrefix(section, "profile "):
			name := strings.TrimSpace(strings.TrimPrefix(section, "profile "))
			if name != "" {
				seen[name] = struct{}{}
			}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	// Stable order: default first (if present), then others alphabetically.
	var out []string
	if _, ok := seen["default"]; ok {
		out = append(out, "default")
		delete(seen, "default")
	}

	others := make([]string, 0, len(seen))
	for k := range seen {
		others = append(others, k)
	}
	sort.Strings(others)
	out = append(out, others...)

	return out
}
