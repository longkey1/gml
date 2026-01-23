package gml

import (
	"fmt"
	"strings"
)

// LabelIndex provides fast lookup for label names and IDs
type LabelIndex struct {
	nameToID map[string]string
	idToName map[string]string
	idToID   map[string]string
}

// FetchLabelIndex fetches all labels and builds an index for fast lookup
func FetchLabelIndex(svc *Service) (*LabelIndex, error) {
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

	return &LabelIndex{
		nameToID: nameToID,
		idToName: idToName,
		idToID:   idToID,
	}, nil
}

// ResolveLabelIDs converts label names or IDs to valid label IDs
// Supports both system labels (INBOX, SENT) and custom labels
func (idx *LabelIndex) ResolveLabelIDs(requested []string) ([]string, error) {
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

// MapLabelIDsToNames converts label IDs to human-readable names
func (idx *LabelIndex) MapLabelIDsToNames(ids []string) []string {
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

// GetUserEmail retrieves the authenticated user's email address
func GetUserEmail(svc *Service) (string, error) {
	profile, err := svc.Gmail.Users.GetProfile("me").Do()
	if err != nil {
		return "", fmt.Errorf("unable to get user profile: %w", err)
	}
	return profile.EmailAddress, nil
}

// BuildMailURL constructs a Gmail web UI URL for a thread
func BuildMailURL(email, threadID string) string {
	// Note: url.QueryEscape is not needed here as email addresses don't need escaping
	return fmt.Sprintf("https://mail.google.com/mail/?authuser=%s#all/%s", email, threadID)
}
