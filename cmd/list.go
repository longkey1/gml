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

const defaultFields = "id,from,subject,date,labels,snippet"

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Gmail messages",
	Long: `List Gmail messages with optional filters.

Available fields: id, threadid, url, from, to, subject, date, labels, snippet, body

Common labels: INBOX, SENT, DRAFT, SPAM, TRASH, STARRED, UNREAD, IMPORTANT,
               CATEGORY_PERSONAL, CATEGORY_SOCIAL, CATEGORY_PROMOTIONS,
               CATEGORY_UPDATES, CATEGORY_FORUMS

Examples:
  gml list                              # List recent messages
  gml list -q "from:example@gmail.com"  # Search messages
  gml list -n 20                        # Get 20 messages
  gml list -l INBOX                     # List messages in INBOX
  gml list -l INBOX -l UNREAD           # List unread messages in INBOX
  gml list -f id,from,subject,body      # Specify fields to include
  gml list --format json                # Output as JSON`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := GetConfig()

	// Get flags
	query, _ := cmd.Flags().GetString("query")
	maxResults, _ := cmd.Flags().GetInt64("max-results")
	labels, _ := cmd.Flags().GetStringArray("label")
	format, _ := cmd.Flags().GetString("format")
	fieldsStr, _ := cmd.Flags().GetString("fields")

	// Parse fields
	fields := gml.ParseFields(fieldsStr)

	// Create service
	svc, err := gml.NewService(ctx, cfg)
	if err != nil {
		return fmt.Errorf("unable to create service: %w", err)
	}

	// List messages
	messages, err := gml.ListMessages(ctx, svc, gml.ListMessagesOptions{
		Query:      query,
		MaxResults: maxResults,
		LabelIDs:   labels,
		Fields:     fields,
	})
	if err != nil {
		return fmt.Errorf("unable to list messages: %w", err)
	}

	if len(messages) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No messages found.")
		return nil
	}

	// Output
	outputFormat := gml.OutputFormat(format)
	if err := gml.FormatMessageList(cmd.OutOrStdout(), messages, fields, outputFormat); err != nil {
		return fmt.Errorf("unable to format output: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("query", "q", "", "Search query (Gmail search syntax)")
	listCmd.Flags().Int64P("max-results", "n", 10, "Maximum number of messages to return")
	listCmd.Flags().StringArrayP("label", "l", nil, "Filter by label (can be specified multiple times)")
	listCmd.Flags().String("format", "text", "Output format (text or json)")
	listCmd.Flags().StringP("fields", "f", defaultFields, "Comma-separated list of fields (id,from,to,subject,date,labels,snippet,body)")

	// Set custom output to enable testing
	listCmd.SetOut(os.Stdout)
}
