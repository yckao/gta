package provider

// Options is a marker interface for provider-specific options
type Options interface {
	IsOptions()
}

// Provider defines the interface that all cloud providers must implement
type Provider interface {
	// Grant grants temporary access with the given options
	Grant(opts Options) error

	// Revoke revokes temporary access with the given options
	Revoke(opts Options) error

	// ListTemporaryBindings lists temporary bindings with the given options
	ListTemporaryBindings(opts Options) error

	// CleanTemporaryBindings lists and optionally removes temporary bindings with the given options
	CleanTemporaryBindings(opts Options) error
}
