package gml

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
)

// FormatMessageList outputs messages in the specified format
func FormatMessageList(w io.Writer, messages []MessageInfo, fields map[string]bool, format OutputFormat) error {
	if format == OutputFormatJSON {
		return formatMessagesJSON(w, messages)
	}
	return formatMessagesTable(w, messages, fields)
}

// FormatMessageDetail outputs a message detail in the specified format
func FormatMessageDetail(w io.Writer, detail *MessageDetail, format OutputFormat) error {
	if format == OutputFormatJSON {
		return formatDetailJSON(w, detail)
	}
	return formatDetailText(w, detail)
}

// formatMessagesJSON outputs messages as JSON
func formatMessagesJSON(w io.Writer, messages []MessageInfo) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// formatMessagesTable outputs messages as a table
func formatMessagesTable(w io.Writer, messages []MessageInfo, fields map[string]bool) error {
	// Build header based on selected fields
	var headers []any
	fieldOrder := []string{"id", "threadid", "url", "from", "to", "subject", "date", "labels", "snippet"}
	for _, f := range fieldOrder {
		if fields[f] {
			headers = append(headers, strings.ToUpper(f))
		}
	}

	table := tablewriter.NewWriter(w)
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
			case "threadid":
				row = append(row, msg.ThreadID)
			case "url":
				row = append(row, msg.URL)
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
		if err := table.Append(row); err != nil {
			return err
		}
	}

	if err := table.Render(); err != nil {
		return err
	}

	// Print body separately if requested
	if fields["body"] {
		for _, msg := range messages {
			if msg.Body != "" {
				if _, err := fmt.Fprintf(w, "\n=== %s ===\n%s\n", msg.ID, msg.Body); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// formatDetailJSON outputs message detail as JSON
func formatDetailJSON(w io.Writer, detail *MessageDetail) error {
	data, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// formatDetailText outputs message detail as text
func formatDetailText(w io.Writer, detail *MessageDetail) error {
	lines := []string{
		fmt.Sprintf("ID: %s", detail.ID),
		fmt.Sprintf("ThreadID: %s", detail.ThreadID),
		fmt.Sprintf("URL: %s", detail.URL),
		fmt.Sprintf("From: %s", detail.From),
		fmt.Sprintf("To: %s", detail.To),
		fmt.Sprintf("Subject: %s", detail.Subject),
		fmt.Sprintf("Date: %s", detail.Date),
	}
	if len(detail.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("Labels: %s", strings.Join(detail.Labels, ", ")))
	}
	lines = append(lines, "---", detail.Body)

	_, err := fmt.Fprintln(w, strings.Join(lines, "\n"))
	return err
}

// truncate truncates a string to maxLen with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
