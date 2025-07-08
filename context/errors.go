package context

import "errors"

var (
	// ErrConfigRequired is returned when no configuration is provided
	ErrConfigRequired = errors.New("configuration is required")

	// ErrNoConfigProvider is returned when trying to reload without a provider
	ErrNoConfigProvider = errors.New("no configuration provider available for reload")

	// ErrNilClient is returned when a required client is nil
	ErrNilClient = errors.New("client is nil")
)