package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Client wraps AWS SSM client with profile information
type Client struct {
	ssmClient *ssm.Client
	profile   string
}

// NewClient creates an AWS SSM client for the specified profile
func NewClient(ctx context.Context, profile string) (*Client, error) {
	return NewClientWithRegion(ctx, profile, "")
}

// NewClientWithRegion creates an AWS SSM client for the specified profile with optional region override
func NewClientWithRegion(ctx context.Context, profile, region string) (*Client, error) {
	var cfg aws.Config
	var err error

	// Build config options
	var opts []func(*config.LoadOptions) error

	// If profile is not "default", add profile option
	if profile != "default" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	// If region is specified, add region option
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	// Load config with options
	cfg, err = config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for profile %s: %w", profile, err)
	}

	return &Client{
		ssmClient: ssm.NewFromConfig(cfg),
		profile:   profile,
	}, nil
}

// NewClientPool creates AWS clients for multiple profiles
func NewClientPool(ctx context.Context, profiles []string) (map[string]*Client, error) {
	return NewClientPoolWithRegion(ctx, profiles, "")
}

// NewClientPoolWithRegion creates AWS clients for multiple profiles with optional region override
func NewClientPoolWithRegion(ctx context.Context, profiles []string, region string) (map[string]*Client, error) {
	pool := make(map[string]*Client, len(profiles))

	for _, profile := range profiles {
		client, err := NewClientWithRegion(ctx, profile, region)
		if err != nil {
			return nil, fmt.Errorf("failed to create client for profile %s: %w", profile, err)
		}
		pool[profile] = client
	}

	return pool, nil
}

// Profile returns the profile name for this client
func (c *Client) Profile() string {
	return c.profile
}
