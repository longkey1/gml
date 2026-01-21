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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/longkey1/gml/internal/gml"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
)

var (
	getFormat string
)

// MessageDetail represents a full message with body for output
type MessageDetail struct {
	ID       string   `json:"id"`
	ThreadID string   `json:"threadId"`
	URL      string   `json:"url"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Subject  string   `json:"subject"`
	Date     string   `json:"date"`
	Labels   []string `json:"labels"`
	Body     string   `json:"body"`
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <message-id>",
	Short: "Get a Gmail message with full body",
	Long: `Get a Gmail message by ID with full body content.

Examples:
  gml get 18abc123def456    # Get message by ID
  gml get 18abc123def456 --format json  # Output as JSON`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		messageID := args[0]

		ctx := context.Background()
		cfg := GetConfig()

		svc, err := gml.NewService(ctx, cfg)
		if err != nil {
			log.Fatalf("Unable to create service: %v", err)
		}

		profile, err := svc.Gmail.Users.GetProfile("me").Do()
		if err != nil {
			log.Fatalf("Unable to get user profile: %v", err)
		}

		labelsIndex, err := fetchLabelIndex(svc)
		if err != nil {
			log.Fatalf("Unable to fetch labels: %v", err)
		}

		msg, err := svc.Gmail.Users.Messages.Get("me", messageID).Format("full").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve message: %v", err)
		}

		detail := MessageDetail{
			ID:       msg.Id,
			ThreadID: msg.ThreadId,
			URL:      buildMailURL(profile.EmailAddress, msg.ThreadId),
			Labels:   mapLabelIDsToNames(msg.LabelIds, labelsIndex),
		}

		for _, header := range msg.Payload.Headers {
			switch header.Name {
			case "From":
				detail.From = header.Value
			case "To":
				detail.To = header.Value
			case "Subject":
				detail.Subject = header.Value
			case "Date":
				detail.Date = header.Value
			}
		}

		detail.Body = extractBody(msg.Payload)

		if getFormat == "json" {
			outputDetailJSON(detail)
		} else {
			outputDetailText(detail)
		}
	},
}

// extractBody extracts the message body from payload
func extractBody(payload *gmail.MessagePart) string {
	if payload == nil {
		return ""
	}

	// Try to get plain text body first
	body := findBodyPart(payload, "text/plain")
	if body != "" {
		return body
	}

	// Fall back to HTML body
	body = findBodyPart(payload, "text/html")
	if body != "" {
		return body
	}

	// If no parts, try the main body
	if payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err != nil {
			return ""
		}
		return string(decoded)
	}

	return ""
}

// findBodyPart recursively finds a body part with the specified MIME type
func findBodyPart(part *gmail.MessagePart, mimeType string) string {
	if part.MimeType == mimeType && part.Body != nil && part.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err != nil {
			return ""
		}
		return string(decoded)
	}

	for _, p := range part.Parts {
		if body := findBodyPart(p, mimeType); body != "" {
			return body
		}
	}

	return ""
}

func outputDetailJSON(detail MessageDetail) {
	data, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		log.Fatalf("Unable to marshal JSON: %v", err)
	}
	fmt.Println(string(data))
}

func outputDetailText(detail MessageDetail) {
	fmt.Printf("ID: %s\n", detail.ID)
	fmt.Printf("ThreadID: %s\n", detail.ThreadID)
	fmt.Printf("URL: %s\n", detail.URL)
	fmt.Printf("From: %s\n", detail.From)
	fmt.Printf("To: %s\n", detail.To)
	fmt.Printf("Subject: %s\n", detail.Subject)
	fmt.Printf("Date: %s\n", detail.Date)
	if len(detail.Labels) > 0 {
		fmt.Printf("Labels: %s\n", strings.Join(detail.Labels, ", "))
	}
	fmt.Println("---")
	fmt.Println(detail.Body)
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringVar(&getFormat, "format", "text", "Output format (text or json)")
}
