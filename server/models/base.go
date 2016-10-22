package models

import "strings"

// Object is base of object models
type Object struct {
	// OID is object id
	OID string `json:"oid"`
}

// Link is a location reference a remote object
type Link struct {
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

// Namespace represents a hierarchy
type Namespace []string

// String returns the string representative of the namespace
// joining components by /
func (ns Namespace) String() string {
	return strings.Join(ns, "/")
}

// FQN returns full-qualified-name of the name prefixed by
// the namespace
func (ns Namespace) FQN(name string) string {
	return ns.String() + "/" + name
}

// Properties defines a generic set of properties
type Properties map[string]interface{}

// PropertyNS is a flat namespace of multiple Properties
type PropertyNS map[string]Properties
