package google

import (
	"context"
	"fmt"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailService wraps the Google Gmail API service
type GmailService struct {
	*gmail.Service
}

// NewGmailService creates a new Gmail service with the given authenticator
func NewGmailService(ctx context.Context, auth Authenticator) (*GmailService, error) {
	client, err := auth.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %v", err)
	}

	var srv *gmail.Service
	if client != nil {
		srv, err = gmail.NewService(ctx, option.WithHTTPClient(client))
	} else {
		// Use Application Default Credentials (for Service Account)
		srv, err = gmail.NewService(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %v", err)
	}

	return &GmailService{srv}, nil
}
