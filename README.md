# PS9S - AWS Parameter Store TUI

A beautiful terminal user interface (TUI) for managing AWS Systems Manager Parameter Store parameters across multiple AWS profiles.

## Features

- **Multi-Profile Support**: Seamlessly switch between multiple AWS profiles and regions
- **Recent Contexts**: Remembers your last 5 profile/region combinations for quick switching (1-5 keys)
- **Smart Navigation**: Press 'p' to jump directly to profile selection from any parameter list
- **Interactive UI**: Built with Bubble Tea for a smooth terminal experience
- **Search & Filter**: Quickly find parameters with real-time search
- **View & Edit**: View parameter details and edit values inline
- **JSON Support**: View and edit individual JSON keys within parameter values
- **Copy to Clipboard**: Press 'c' to copy values to your system clipboard
- **SecureString Support**: Automatically decrypts SecureString parameters (requires KMS permissions)
- **All Parameter Types**: Supports String, StringList, and SecureString types
- **Context Persistence**: Automatically remembers your last selected region for each profile

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/caseycs/ps9s.git
cd ps9s

# Build the binary
go build -o ps9s ./cmd/ps9s

# Move to a location in your PATH (optional)
sudo mv ps9s /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/caseycs/ps9s/cmd/ps9s@latest
```

## Prerequisites

1. **AWS Credentials**: Ensure your AWS credentials are configured
   ```bash
   aws configure
   ```

2. **AWS Profiles**: Set up your AWS profiles in `~/.aws/config`
   ```ini
   [profile dev]
   region = us-east-1

   [profile staging]
   region = us-west-2

   [profile prod]
   region = us-east-1
   ```

3. **IAM Permissions**: Your AWS user/role needs the following permissions:
   - `ssm:DescribeParameters`
   - `ssm:GetParameter`
   - `ssm:PutParameter`
   - `kms:Decrypt` (for SecureString parameters)

## Usage

### Quick Start

Simply run the application:

```bash
ps9s
```

By default, it will use your current AWS profile (from `AWS_PROFILE` environment variable) or the `default` profile.

### Multi-Profile Mode (Optional)

To use multiple AWS profiles, set the `PS9S_AWS_PROFILES` environment variable:

```bash
export PS9S_AWS_PROFILES="dev,staging,prod"
```

Add this to your `~/.bashrc`, `~/.zshrc`, or `~/.profile` to make it permanent.

When `PS9S_AWS_PROFILES` is set, the application will show a profile selector on startup.

### Using Environment Variables for AWS Credentials

If you don't have AWS profiles configured, you can use environment variables:

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"  # Region is required!

ps9s
```

**Important**: The `AWS_REGION` (or `AWS_DEFAULT_REGION`) environment variable is **required** when using credentials via environment variables.

## Navigation

### Profile Selector Screen
- **↑/↓ or j/k**: Navigate through profiles
- **Enter**: Select a profile
- **Esc**: No action (prevents accidental quit)
- **q or Ctrl+C**: Quit

### Region Selector Screen
- **↑/↓ or j/k**: Navigate through regions
- **Enter**: Select a region
- **Esc**: Go back to profile selector
- **q or Ctrl+C**: Quit

### Parameter List Screen
- **↑/↓ or j/k**: Navigate through parameters
- **Enter**: View selected parameter
- **/**: Activate search mode
- **p**: Jump to profile selection
- **1-5**: Quick switch to recent profile/region contexts
- **Esc**: Go back to region selector
- **q or Ctrl+C**: Quit

### Search Mode
- **Type**: Filter parameters by name (case-insensitive)
- **Enter**: Apply filter and exit search mode
- **Esc**: Clear filter and exit search mode

### Parameter View Screen
- **↑/↓ or j/k**: Navigate through JSON keys (if parameter is JSON)
- **e**: Edit parameter value (or selected JSON key)
- **c**: Copy value to clipboard (whole parameter or selected JSON key)
- **Esc**: Go back to parameter list
- **q or Ctrl+C**: Quit

### Parameter Edit Screen
- **Type**: Edit the parameter value (multi-line supported)
- **Ctrl+S**: Save changes to AWS
- **Esc**: Cancel and go back without saving
- **Ctrl+C**: Quit

## Architecture

```
ps9s/
├── cmd/ps9s/
│   └── main.go              # Application entry point
├── internal/
│   ├── aws/
│   │   ├── client.go        # AWS SSM client wrapper
│   │   └── parameter.go     # Parameter operations
│   ├── config/
│   │   └── config.go        # Environment variable parsing
│   └── ui/
│       ├── model.go         # Root orchestrator model
│       ├── messages.go      # Navigation messages
│       ├── styles.go        # Lipgloss styles
│       └── screens/
│           ├── profile_selector.go
│           ├── parameter_list.go
│           ├── parameter_view.go
│           └── parameter_edit.go
```

## Configuration

### Persistent Configuration

PS9S stores configuration in `~/.ps9s/`:
- `recents.json` - Last 5 profile/region combinations for quick switching
- `regions.json` - Last selected region for each profile

### Environment Variables

- `PS9S_AWS_PROFILES` (optional): Comma-separated list of AWS profile names for multi-profile mode
  ```bash
  export PS9S_AWS_PROFILES="profile1,profile2,profile3"
  ```
  If not set, uses the current `AWS_PROFILE` or `default` profile.

- Standard AWS environment variables are also respected:
  - `AWS_PROFILE` - The current AWS profile to use (when `PS9S_AWS_PROFILES` is not set)
  - `AWS_REGION` or `AWS_DEFAULT_REGION` - **Required** when using environment variables for credentials
  - `AWS_ACCESS_KEY_ID` - Access key for authentication
  - `AWS_SECRET_ACCESS_KEY` - Secret key for authentication
  - `AWS_SESSION_TOKEN` - Session token (for temporary credentials)

## Error Handling

The application provides user-friendly error messages for common issues:

- **Invalid Profile**: Shows which profile failed to load
- **Access Denied**: Indicates missing AWS permissions
- **Parameter Not Found**: Notifies when a parameter has been deleted
- **KMS Errors**: Shows clear message when SecureString decryption fails

## Examples

### Viewing Parameters (Single Profile)

1. Run ps9s (uses current AWS profile):
   ```bash
   ps9s
   ```

2. Browse parameters, use `/` to search

3. Press Enter to view a parameter's details

### Viewing Parameters (Multi-Profile)

1. Set your profiles:
   ```bash
   export PS9S_AWS_PROFILES="dev,prod"
   ```

2. Run ps9s:
   ```bash
   ps9s
   ```

3. Select a profile using arrow keys and Enter

4. Browse parameters, use `/` to search

5. Press Enter to view a parameter's details

### Editing a Parameter

1. Navigate to a parameter and press Enter to view it

2. Press `e` to enter edit mode
   - For JSON parameters, use ↑/↓ to select a specific key, then press `e` to edit just that key
   - For regular parameters, `e` edits the entire value

3. Modify the value

4. Press `Ctrl+S` to save changes to AWS

5. The application will return to the view screen with updated values

### Using Recent Contexts

1. After viewing parameters from different profile/region combinations, they're saved to recent contexts

2. From the parameter list screen, press `1-5` to quickly switch between your last 5 contexts

3. The current context is dimmed in the recent list

4. Recent contexts are preserved across app restarts (saved to `~/.ps9s/recents.json`)

## Troubleshooting

### "failed to load AWS config for profile X"

Ensure the profile exists in `~/.aws/config`:
```bash
aws configure --profile X
```

### "Access denied" errors

Check your IAM permissions. You need:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:DescribeParameters",
        "ssm:GetParameter",
        "ssm:PutParameter"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt"
      ],
      "Resource": "*"
    }
  ]
}
```

### Cannot decrypt SecureString parameters

Ensure you have KMS decrypt permissions for the KMS key used to encrypt the parameters.

## Development

### Building

```bash
go build -o ps9s ./cmd/ps9s
```

### Running Tests

```bash
go test ./...
```

### Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) - AWS integration

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

Built with ❤️ using [Bubble Tea](https://github.com/charmbracelet/bubbletea)
