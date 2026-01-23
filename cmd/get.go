/*
Copyright © 2025 longkey1

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/longkey1/gml/internal/gml"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <message-id>",
	Short: "Get a Gmail message with full body",
	Long: `Get a Gmail message by ID with full body content.

Examples:
  gml get 18abc123def456    # Get message by ID
  gml get 18abc123def456 --format json  # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	ctx := cmd.Context()
	cfg := GetConfig()

	// Get flags
	format, _ := cmd.Flags().GetString("format")

	// Create service
	svc, err := gml.NewService(ctx, cfg)
	if err != nil {
		return fmt.Errorf("unable to create service: %w", err)
	}

	// Get message
	detail, err := gml.GetMessage(ctx, svc, messageID)
	if err != nil {
		return fmt.Errorf("unable to get message: %w", err)
	}

	// Output
	outputFormat := gml.OutputFormat(format)
	if err := gml.FormatMessageDetail(cmd.OutOrStdout(), detail, outputFormat); err != nil {
		return fmt.Errorf("unable to format output: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().String("format", "text", "Output format (text or json)")

	// Set custom output to enable testing
	getCmd.SetOut(os.Stdout)
}
