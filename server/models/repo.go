package models

// Repository represents source repository
type Repository struct {
	Object
	// Name is the short name of the repository
	Name string `json:"name"`
	// Namespace defines the hierarchy of the repo
	// e.g. for github, Namespace only contains one
	// component which is organization
	Namespace Namespace `json:"namespace"`
	// SCM is the name of source control manager, e.g. github
	SCM string `json:"scm"`
	// Links is the list of URLs to access the repository
	Links []Link `json:"links"`
	// Properties contains extra properties in namespace
	// e.g. SCM can store properties in namespace under SCM name
	Properties PropertyNS `json:"properties"`
}
