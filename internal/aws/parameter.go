package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// Parameter represents an AWS Systems Manager parameter
type Parameter struct {
	Name             string
	Type             string
	Value            string
	ARN              string
	Version          int64
	LastModifiedDate time.Time
	DataType         string
}

// ListParameters retrieves all parameters for the profile with pagination
func (c *Client) ListParameters(ctx context.Context) ([]*Parameter, error) {
	var parameters []*Parameter
	var nextToken *string

	for {
		input := &ssm.DescribeParametersInput{
			MaxResults: aws.Int32(50), // Max allowed by AWS
			NextToken:  nextToken,
		}

		output, err := c.ssmClient.DescribeParameters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe parameters: %w", err)
		}

		for _, p := range output.Parameters {
			param := &Parameter{
				Name:             aws.ToString(p.Name),
				Type:             string(p.Type),
				Version:          p.Version,
				LastModifiedDate: aws.ToTime(p.LastModifiedDate),
			}
			if p.ARN != nil {
				param.ARN = aws.ToString(p.ARN)
			}
			if p.DataType != nil {
				param.DataType = aws.ToString(p.DataType)
			}
			parameters = append(parameters, param)
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}

	return parameters, nil
}

// GetParameter retrieves a specific parameter with its value (decrypted if SecureString)
func (c *Client) GetParameter(ctx context.Context, name string) (*Parameter, error) {
	withDecryption := true

	output, err := c.ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(withDecryption),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get parameter %s: %w", name, err)
	}

	p := output.Parameter
	param := &Parameter{
		Name:             aws.ToString(p.Name),
		Type:             string(p.Type),
		Value:            aws.ToString(p.Value),
		ARN:              aws.ToString(p.ARN),
		Version:          p.Version,
		LastModifiedDate: aws.ToTime(p.LastModifiedDate),
	}
	if p.DataType != nil {
		param.DataType = aws.ToString(p.DataType)
	}

	return param, nil
}

// PutParameter updates a parameter's value
func (c *Client) PutParameter(ctx context.Context, name, value, paramType string) error {
	// Use Overwrite to update existing parameter
	overwrite := true

	input := &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      types.ParameterType(paramType),
		Overwrite: aws.Bool(overwrite),
	}

	_, err := c.ssmClient.PutParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put parameter %s: %w", name, err)
	}

	return nil
}
