package gml

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// MessageInfo represents a simplified message for output
type MessageInfo struct {
	ID       string   `json:"id,omitempty"`
	ThreadID string   `json:"threadId,omitempty"`
	URL      string   `json:"url,omitempty"`
	From     string   `json:"from,omitempty"`
	To       string   `json:"to,omitempty"`
	Subject  string   `json:"subject,omitempty"`
	Date     string   `json:"date,omitempty"`
	Snippet  string   `json:"snippet,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	Body     string   `json:"body,omitempty"`
}

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

// ListMessagesOptions contains options for listing messages
type ListMessagesOptions struct {
	Query      string
	MaxResults int64
	LabelIDs   []string
	Fields     map[string]bool
}

// ListMessages fetches messages with pagination and returns message info
func ListMessages(ctx context.Context, svc *Service, opts ListMessagesOptions) ([]MessageInfo, error) {
	// Fetch user email if URL field is requested
	var userEmail string
	if opts.Fields["url"] {
		email, err := GetUserEmail(svc)
		if err != nil {
			return nil, err
		}
		userEmail = email
	}

	// Fetch label mappings if needed
	var labelsIndex *LabelIndex
	if len(opts.LabelIDs) > 0 || opts.Fields["labels"] {
		idx, err := FetchLabelIndex(svc)
		if err != nil {
			return nil, err
		}
		labelsIndex = idx
	}

	// Resolve label names to IDs if needed
	resolvedLabels := opts.LabelIDs
	if len(opts.LabelIDs) > 0 && labelsIndex != nil {
		labels, err := labelsIndex.ResolveLabelIDs(opts.LabelIDs)
		if err != nil {
			return nil, err
		}
		resolvedLabels = labels
	}

	// List messages with pagination
	var allMessages []*gmail.Message
	pageToken := ""

	for {
		call := svc.Gmail.Users.Messages.List("me").MaxResults(opts.MaxResults).Context(ctx)
		if opts.Query != "" {
			call = call.Q(opts.Query)
		}
		if len(resolvedLabels) > 0 {
			call = call.LabelIds(resolvedLabels...)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve messages: %w", err)
		}

		allMessages = append(allMessages, result.Messages...)

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	if len(allMessages) == 0 {
		return nil, nil
	}

	// Determine if we need full format (for body)
	needsBody := opts.Fields["body"]

	// Get message details
	var messages []MessageInfo
	for _, m := range allMessages {
		var msg *gmail.Message
		var err error

		if needsBody {
			msg, err = svc.Gmail.Users.Messages.Get("me", m.Id).Format("full").Context(ctx).Do()
		} else {
			msg, err = svc.Gmail.Users.Messages.Get("me", m.Id).Format("metadata").
				MetadataHeaders("From", "To", "Subject", "Date").Context(ctx).Do()
		}
		if err != nil {
			// Skip messages we can't retrieve instead of failing completely
			continue
		}

		info := buildMessageInfo(msg, opts.Fields, userEmail, labelsIndex)

		if needsBody {
			info.Body = ExtractBody(msg.Payload)
		}

		messages = append(messages, info)
	}

	return messages, nil
}

// GetMessage retrieves a single message by ID with full details
func GetMessage(ctx context.Context, svc *Service, messageID string) (*MessageDetail, error) {
	userEmail, err := GetUserEmail(svc)
	if err != nil {
		return nil, err
	}

	labelsIndex, err := FetchLabelIndex(svc)
	if err != nil {
		return nil, err
	}

	msg, err := svc.Gmail.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve message: %w", err)
	}

	detail := &MessageDetail{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		URL:      BuildMailURL(userEmail, msg.ThreadId),
		Labels:   labelsIndex.MapLabelIDsToNames(msg.LabelIds),
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

	detail.Body = ExtractBody(msg.Payload)

	return detail, nil
}

// buildMessageInfo constructs a MessageInfo from a Gmail message
func buildMessageInfo(msg *gmail.Message, fields map[string]bool, userEmail string, labelsIndex *LabelIndex) MessageInfo {
	info := MessageInfo{}

	if fields["id"] {
		info.ID = msg.Id
	}
	if fields["threadid"] {
		info.ThreadID = msg.ThreadId
	}
	if fields["url"] {
		info.URL = BuildMailURL(userEmail, msg.ThreadId)
	}
	if fields["labels"] && labelsIndex != nil {
		info.Labels = labelsIndex.MapLabelIDsToNames(msg.LabelIds)
	}
	if fields["snippet"] {
		info.Snippet = msg.Snippet
	}

	if msg.Payload != nil {
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
	}

	return info
}

// ExtractBody extracts the message body from payload
func ExtractBody(payload *gmail.MessagePart) string {
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

// ParseFields parses a comma-separated field string into a map
func ParseFields(fieldsStr string) map[string]bool {
	fields := make(map[string]bool)
	for _, f := range strings.Split(fieldsStr, ",") {
		fields[strings.TrimSpace(strings.ToLower(f))] = true
	}
	return fields
}
