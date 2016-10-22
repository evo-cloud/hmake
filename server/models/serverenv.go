package models

// ServerEnv represents the server runtime environment
type ServerEnv struct {
	// SCMProviders is the list of registered SCMProviders
	SCMProviders []string `json:"scm-providers"`
}
