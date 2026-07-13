package gml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// writeConfigFile creates a TOML config file and points the global
// viper instance at it.
func writeConfigFile(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("ReadInConfig() error = %v", err)
	}
}

// LoadConfig reads from the global viper instance, so these tests
// reset it per subtest and do not run in parallel.
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Config
	}{
		{
			name: "all fields",
			content: `
auth_type = "service_account"
application_credentials = "/path/to/credentials.json"
user_credentials = "/path/to/token.json"
`,
			want: Config{
				AuthType:                     AuthTypeServiceAccount,
				GoogleApplicationCredentials: "/path/to/credentials.json",
				GoogleUserCredentials:        "/path/to/token.json",
			},
		},
		{
			name: "oauth explicit",
			content: `
auth_type = "oauth"
application_credentials = "/path/to/credentials.json"
user_credentials = "/path/to/token.json"
`,
			want: Config{
				AuthType:                     AuthTypeOAuth,
				GoogleApplicationCredentials: "/path/to/credentials.json",
				GoogleUserCredentials:        "/path/to/token.json",
			},
		},
		{
			name: "missing auth_type defaults to oauth",
			content: `
application_credentials = "/path/to/credentials.json"
user_credentials = "/path/to/token.json"
`,
			want: Config{
				AuthType:                     AuthTypeOAuth,
				GoogleApplicationCredentials: "/path/to/credentials.json",
				GoogleUserCredentials:        "/path/to/token.json",
			},
		},
		{
			name:    "empty file defaults to oauth",
			content: "",
			want: Config{
				AuthType: AuthTypeOAuth,
			},
		},
		{
			name: "unknown keys are ignored",
			content: `
auth_type = "oauth"
application_credentials = "/path/to/credentials.json"
user_credentials = "/path/to/token.json"
unknown_key = "ignored"
`,
			want: Config{
				AuthType:                     AuthTypeOAuth,
				GoogleApplicationCredentials: "/path/to/credentials.json",
				GoogleUserCredentials:        "/path/to/token.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			writeConfigFile(t, tt.content)

			got, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			if *got != tt.want {
				t.Errorf("LoadConfig() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestLoadConfigWithoutFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	want := Config{AuthType: AuthTypeOAuth}
	if *got != want {
		t.Errorf("LoadConfig() = %+v, want %+v", *got, want)
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid oauth",
			config: Config{
				AuthType:                     AuthTypeOAuth,
				GoogleApplicationCredentials: "/path/to/credentials.json",
				GoogleUserCredentials:        "/path/to/token.json",
			},
		},
		{
			name: "valid service account without user credentials",
			config: Config{
				AuthType:                     AuthTypeServiceAccount,
				GoogleApplicationCredentials: "/path/to/credentials.json",
			},
		},
		{
			name: "missing application credentials",
			config: Config{
				AuthType:              AuthTypeOAuth,
				GoogleUserCredentials: "/path/to/token.json",
			},
			wantErr: true,
		},
		{
			name: "oauth without user credentials",
			config: Config{
				AuthType:                     AuthTypeOAuth,
				GoogleApplicationCredentials: "/path/to/credentials.json",
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
