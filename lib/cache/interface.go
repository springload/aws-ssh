package cache

import (
	"aws-ssh/lib"
)

// Cache represents the cache for profiles
type Cache interface {
	// Load() loads the cache
	Load() ([]lib.ProcessedProfileSummary, error)
	// Save() saves the cache
	Save([]lib.ProcessedProfileSummary) error
	// Lookup looks up ssh entry by its name
	// If name is empty or there is no exact match,
	// it switches to the fuzzy search mode
	Lookup(name string) (lib.SSHEntry, error)
	// ListCanonicalNames() returns all known canonical host names from the cache
	ListCanonicalNames() ([]string, error)
}
