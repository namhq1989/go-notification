package notification

import "fmt"

// Client is the main notification client.
// Use New() to create, callers store as Notification interface.
type Client struct {
	firebase firebaseSender
	enabled  bool
}

// New creates a new notification client.
// If Enabled is false, client operates in dev mode (log only, no FCM calls).
func New(cfg Config) (*Client, error) {
	if !cfg.Enabled {
		return &Client{enabled: false}, nil
	}

	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("%s: project ID is required", KeyNotificationMissingConfig)
	}

	if !hasCredentials(cfg) {
		return nil, fmt.Errorf("%s: credentials (file or base64) required", KeyNotificationMissingConfig)
	}

	fb, err := NewFirebaseClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		firebase: fb,
		enabled:  true,
	}, nil
}

// hasCredentials returns true if credentials are configured.
// If both are set, base64 wins (documented in Config).
func hasCredentials(cfg Config) bool {
	return cfg.CredentialsFile != "" || cfg.CredentialsBase64 != ""
}
