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
)

var (
	listQuery      string
	listMaxResults int64
	listUnread     bool
	listFormat     string
)

// MessageInfo represents a simplified message for output
type MessageInfo struct {
	ID      string   `json:"id"`
	From    string   `json:"from"`
	Subject string   `json:"subject"`
	Date    string   `json:"date"`
	Snippet string   `json:"snippet"`
	Labels  []string `json:"labels"`
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Gmail messages",
	Long: `List Gmail messages with optional filters.

Examples:
  gml list                      # List recent messages
  gml list -u                   # List unread messages
  gml list -q "from:example@gmail.com"  # Search messages
  gml list -n 20                # Get 20 messages
  gml list --format json        # Output as JSON`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		cfg := GetConfig()

		svc, err := gml.NewService(ctx, cfg)
		if err != nil {
			log.Fatalf("Unable to create service: %v", err)
		}

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

		// Get message details
		var messages []MessageInfo
		for _, m := range result.Messages {
			msg, err := svc.Gmail.Users.Messages.Get("me", m.Id).Format("metadata").
				MetadataHeaders("From", "Subject", "Date").Do()
			if err != nil {
				log.Printf("Unable to retrieve message %s: %v", m.Id, err)
				continue
			}

			info := MessageInfo{
				ID:      msg.Id,
				Snippet: msg.Snippet,
				Labels:  msg.LabelIds,
			}

			for _, header := range msg.Payload.Headers {
				switch header.Name {
				case "From":
					info.From = header.Value
				case "Subject":
					info.Subject = header.Value
				case "Date":
					info.Date = header.Value
				}
			}

			messages = append(messages, info)
		}

		// Output
		if listFormat == "json" {
			outputJSON(messages)
		} else {
			outputText(messages)
		}
	},
}

func outputJSON(messages []MessageInfo) {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		log.Fatalf("Unable to marshal JSON: %v", err)
	}
	fmt.Println(string(data))
}

func outputText(messages []MessageInfo) {
	for _, msg := range messages {
		fmt.Printf("ID: %s\n", msg.ID)
		fmt.Printf("From: %s\n", msg.From)
		fmt.Printf("Subject: %s\n", msg.Subject)
		fmt.Printf("Date: %s\n", msg.Date)
		if len(msg.Labels) > 0 {
			fmt.Printf("Labels: %s\n", strings.Join(msg.Labels, ", "))
		}
		fmt.Printf("Snippet: %s\n", msg.Snippet)
		fmt.Println("---")
	}
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listQuery, "query", "q", "", "Search query (Gmail search syntax)")
	listCmd.Flags().Int64VarP(&listMaxResults, "max-results", "n", 10, "Maximum number of messages to return")
	listCmd.Flags().BoolVarP(&listUnread, "unread", "u", false, "Show only unread messages")
	listCmd.Flags().StringVar(&listFormat, "format", "text", "Output format (text or json)")
}
