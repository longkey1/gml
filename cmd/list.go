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
	"os"
	"strings"

	"github.com/longkey1/gml/internal/gml"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
)

var (
	listQuery      string
	listMaxResults int64
	listFormat     string
	listFields     string
	listLabels     []string
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
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		cfg := GetConfig()

		svc, err := gml.NewService(ctx, cfg)
		if err != nil {
			log.Fatalf("Unable to create service: %v", err)
		}

		// Parse fields
		fields := parseFields(listFields)

		// Fetch label mappings if we need to resolve or display labels
		var labelsIndex *labelIndex
		if len(listLabels) > 0 || fields["labels"] {
			idx, err := fetchLabelIndex(svc)
			if err != nil {
				log.Fatalf("Unable to fetch labels: %v", err)
			}
			labelsIndex = idx
		}

		// Build query
		query := listQuery

		// Resolve label names to IDs (supports system labels and custom labels)
		resolvedLabels := listLabels
		if len(listLabels) > 0 {
			labels, err := resolveLabelIDs(labelsIndex, listLabels)
			if err != nil {
				log.Fatalf("Unable to resolve labels: %v", err)
			}
			resolvedLabels = labels
		}

		// List messages with pagination
		var allMessages []*gmail.Message
		pageToken := ""

		for {
			call := svc.Gmail.Users.Messages.List("me").MaxResults(listMaxResults)
			if query != "" {
				call = call.Q(query)
			}
			if len(resolvedLabels) > 0 {
				call = call.LabelIds(resolvedLabels...)
			}
			if pageToken != "" {
				call = call.PageToken(pageToken)
			}

			result, err := call.Do()
			if err != nil {
				log.Fatalf("Unable to retrieve messages: %v", err)
			}

			allMessages = append(allMessages, result.Messages...)

			if result.NextPageToken == "" {
				break
			}
			pageToken = result.NextPageToken
		}

		if len(allMessages) == 0 {
			fmt.Println("No messages found.")
			return
		}

		// Determine if we need full format (for body)
		needsBody := fields["body"]

		// Get message details
		var messages []MessageInfo
		for _, m := range allMessages {
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
				info.Labels = mapLabelIDsToNames(msg.LabelIds, labelsIndex)
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

type labelIndex struct {
	nameToID map[string]string
	idToName map[string]string
	idToID   map[string]string
}

func fetchLabelIndex(svc *gml.Service) (*labelIndex, error) {
	resp, err := svc.Gmail.Users.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list labels: %w", err)
	}

	nameToID := make(map[string]string)
	idToName := make(map[string]string)
	idToID := make(map[string]string)
	for _, l := range resp.Labels {
		nameToID[strings.ToLower(l.Name)] = l.Id
		idToName[strings.ToLower(l.Id)] = l.Name
		idToID[strings.ToLower(l.Id)] = l.Id
	}

	return &labelIndex{
		nameToID: nameToID,
		idToName: idToName,
		idToID:   idToID,
	}, nil
}

func resolveLabelIDs(idx *labelIndex, requested []string) ([]string, error) {
	if idx == nil {
		return nil, fmt.Errorf("label index is nil")
	}

	var resolved []string
	for _, raw := range requested {
		label := strings.ToLower(strings.TrimSpace(raw))
		if id, ok := idx.nameToID[label]; ok {
			resolved = append(resolved, id)
			continue
		}
		if id, ok := idx.idToID[label]; ok {
			resolved = append(resolved, id)
			continue
		}
		return nil, fmt.Errorf("label not found: %s", raw)
	}

	return resolved, nil
}

func mapLabelIDsToNames(ids []string, idx *labelIndex) []string {
	if idx == nil {
		// Fallback to returning IDs as-is
		return ids
	}

	var names []string
	for _, id := range ids {
		if name, ok := idx.idToName[strings.ToLower(id)]; ok {
			names = append(names, name)
		} else {
			names = append(names, id)
		}
	}
	return names
}

func outputJSON(messages []MessageInfo) {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		log.Fatalf("Unable to marshal JSON: %v", err)
	}
	fmt.Println(string(data))
}

func outputText(messages []MessageInfo, fields map[string]bool) {
	// Build header based on selected fields
	var headers []any
	fieldOrder := []string{"id", "from", "to", "subject", "date", "labels", "snippet"}
	for _, f := range fieldOrder {
		if fields[f] {
			headers = append(headers, strings.ToUpper(f))
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header(headers...)

	for _, msg := range messages {
		var row []any
		for _, f := range fieldOrder {
			if !fields[f] {
				continue
			}
			switch f {
			case "id":
				row = append(row, msg.ID)
			case "from":
				row = append(row, truncate(msg.From, 30))
			case "to":
				row = append(row, truncate(msg.To, 30))
			case "subject":
				row = append(row, truncate(msg.Subject, 40))
			case "date":
				row = append(row, msg.Date)
			case "labels":
				row = append(row, strings.Join(msg.Labels, ", "))
			case "snippet":
				row = append(row, truncate(msg.Snippet, 50))
			}
		}
		table.Append(row)
	}

	table.Render()

	// Print body separately if requested
	if fields["body"] {
		for _, msg := range messages {
			if msg.Body != "" {
				fmt.Printf("\n=== %s ===\n%s\n", msg.ID, msg.Body)
			}
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listQuery, "query", "q", "", "Search query (Gmail search syntax)")
	listCmd.Flags().Int64VarP(&listMaxResults, "max-results", "n", 10, "Maximum number of messages to return")
	listCmd.Flags().StringArrayVarP(&listLabels, "label", "l", nil, "Filter by label (can be specified multiple times)")
	listCmd.Flags().StringVar(&listFormat, "format", "text", "Output format (text or json)")
	listCmd.Flags().StringVarP(&listFields, "fields", "f", defaultFields, "Comma-separated list of fields (id,from,to,subject,date,labels,snippet,body)")
}
