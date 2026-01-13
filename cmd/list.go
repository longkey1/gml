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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/longkey1/gml/internal/gml"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
)

var (
	listQuery      string
	listMaxResults int64
	listUnread     bool
	listFormat     string
	listFields     string
)

const defaultFields = "id,from,subject,date,labels,snippet"

// MessageInfo represents a simplified message for output
type MessageInfo struct {
	ID      string   `json:"id,omitempty"`
	From    string   `json:"from,omitempty"`
	To      string   `json:"to,omitempty"`
	Subject string   `json:"subject,omitempty"`
	Date    string   `json:"date,omitempty"`
	Snippet string   `json:"snippet,omitempty"`
	Labels  []string `json:"labels,omitempty"`
	Body    string   `json:"body,omitempty"`
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Gmail messages",
	Long: `List Gmail messages with optional filters.

Available fields: id, from, to, subject, date, labels, snippet, body

Examples:
  gml list                              # List recent messages
  gml list -u                           # List unread messages
  gml list -q "from:example@gmail.com"  # Search messages
  gml list -n 20                        # Get 20 messages
  gml list -f id,from,subject,body      # Specify fields to include
  gml list --format json                # Output as JSON`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		cfg := GetConfig()

		svc, err := gml.NewService(ctx, cfg)
		if err != nil {
			log.Fatalf("Unable to create service: %v", err)
		}

		// Parse fields
		fields := parseFields(listFields)

		// Build query
		query := listQuery
		if listUnread {
			if query != "" {
				query = query + " is:unread"
			} else {
				query = "is:unread"
			}
		}

		// List messages
		call := svc.Gmail.Users.Messages.List("me").MaxResults(listMaxResults)
		if query != "" {
			call = call.Q(query)
		}

		result, err := call.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve messages: %v", err)
		}

		if len(result.Messages) == 0 {
			fmt.Println("No messages found.")
			return
		}

		// Determine if we need full format (for body)
		needsBody := fields["body"]

		// Get message details
		var messages []MessageInfo
		for _, m := range result.Messages {
			var msg *gmail.Message
			var err error

			if needsBody {
				msg, err = svc.Gmail.Users.Messages.Get("me", m.Id).Format("full").Do()
			} else {
				msg, err = svc.Gmail.Users.Messages.Get("me", m.Id).Format("metadata").
					MetadataHeaders("From", "To", "Subject", "Date").Do()
			}
			if err != nil {
				log.Printf("Unable to retrieve message %s: %v", m.Id, err)
				continue
			}

			info := MessageInfo{}

			if fields["id"] {
				info.ID = msg.Id
			}
			if fields["labels"] {
				info.Labels = msg.LabelIds
			}
			if fields["snippet"] {
				info.Snippet = msg.Snippet
			}

			for _, header := range msg.Payload.Headers {
				switch header.Name {
				case "From":
					if fields["from"] {
						info.From = header.Value
					}
				case "To":
					if fields["to"] {
						info.To = header.Value
					}
				case "Subject":
					if fields["subject"] {
						info.Subject = header.Value
					}
				case "Date":
					if fields["date"] {
						info.Date = header.Value
					}
				}
			}

			if needsBody {
				info.Body = extractBody(msg.Payload)
			}

			messages = append(messages, info)
		}

		// Output
		if listFormat == "json" {
			outputJSON(messages)
		} else {
			outputText(messages, fields)
		}
	},
}

func parseFields(fieldsStr string) map[string]bool {
	fields := make(map[string]bool)
	for _, f := range strings.Split(fieldsStr, ",") {
		fields[strings.TrimSpace(strings.ToLower(f))] = true
	}
	return fields
}

func outputJSON(messages []MessageInfo) {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		log.Fatalf("Unable to marshal JSON: %v", err)
	}
	fmt.Println(string(data))
}

func outputText(messages []MessageInfo, fields map[string]bool) {
	for _, msg := range messages {
		if fields["id"] {
			fmt.Printf("ID: %s\n", msg.ID)
		}
		if fields["from"] {
			fmt.Printf("From: %s\n", msg.From)
		}
		if fields["to"] {
			fmt.Printf("To: %s\n", msg.To)
		}
		if fields["subject"] {
			fmt.Printf("Subject: %s\n", msg.Subject)
		}
		if fields["date"] {
			fmt.Printf("Date: %s\n", msg.Date)
		}
		if fields["labels"] && len(msg.Labels) > 0 {
			fmt.Printf("Labels: %s\n", strings.Join(msg.Labels, ", "))
		}
		if fields["snippet"] && msg.Snippet != "" {
			fmt.Printf("Snippet: %s\n", msg.Snippet)
		}
		if fields["body"] && msg.Body != "" {
			fmt.Println("---")
			fmt.Println(msg.Body)
		}
		fmt.Println("---")
	}
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listQuery, "query", "q", "", "Search query (Gmail search syntax)")
	listCmd.Flags().Int64VarP(&listMaxResults, "max-results", "n", 10, "Maximum number of messages to return")
	listCmd.Flags().BoolVarP(&listUnread, "unread", "u", false, "Show only unread messages")
	listCmd.Flags().StringVar(&listFormat, "format", "text", "Output format (text or json)")
	listCmd.Flags().StringVarP(&listFields, "fields", "f", defaultFields, "Comma-separated list of fields (id,from,to,subject,date,labels,snippet,body)")
}
