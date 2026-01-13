package gml

import (
	"context"

	"github.com/longkey1/gml/internal/google"
)

// Service represents the gml application service
type Service struct {
	Gmail *google.GmailService
}

// NewService creates a new gml service based on the configuration
func NewService(ctx context.Context, config *Config) (*Service, error) {
	auth := newAuthenticator(config)

	gmailSvc, err := google.NewGmailService(ctx, auth)
	if err != nil {
		return nil, err
	}

	return &Service{
		Gmail: gmailSvc,
	}, nil
}

func newAuthenticator(config *Config) google.Authenticator {
	switch config.AuthType {
	case AuthTypeServiceAccount:
		return google.NewServiceAccountAuthenticator(config.GoogleApplicationCredentials)
	case AuthTypeOAuth:
		fallthrough
	default:
		return google.NewOAuthAuthenticator(
			config.GoogleApplicationCredentials,
			config.GoogleUserCredentials,
		)
	}
}
