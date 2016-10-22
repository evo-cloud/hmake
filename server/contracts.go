package server

import (
	"io"
	"net/http"
	"net/url"
)

// Logger defines the common logging facility
type Logger interface {
	// With associate context
	With(ctx ...interface{}) Logger

	// log levels

	Fatal(args ...interface{})
	Error(args ...interface{})
	Warn(args ...interface{})
	Info(args ...interface{})
	Debug(args ...interface{})

	Fatalf(fmt string, args ...interface{})
	Errorf(fmt string, args ...interface{})
	Warnf(fmt string, args ...interface{})
	Infof(fmt string, args ...interface{})
	Debugf(fmt string, args ...interface{})
}

// Context is the base context type
type Context interface {
	// GlobalConfig retrieve global configuration
	GlobalConfig() Config

	// GetConfig unmarshals configuration
	GetConfig(interface{}) error

	// Log returns a logger
	Log() Logger

	// Scheduler retrieves task scheduler
	Scheduler() Scheduler
}

// RequestFilter defines a filter before handling requests
type RequestFilter func(w http.ResponseWriter, r *http.Request, next http.Handler)

// InitCtx is the context during initialization
// This is intended to be used for module
type InitCtx interface {
	Context

	// AddSCMProvider registers a source management provider
	AddSCMProvider(name string, provider SCMProvider) error

	// AddHandler registers http.Handler for specified path
	// under current module
	// returns the full url of the endpoint or error
	AddHandler(path string, handler http.Handler) (*url.URL, error)

	// AddRequestFilter registers filter before processing requests
	// useful when plugin authentication/authorization modules
	AddRequestFilter(filter RequestFilter) error
}

// RequestCtx is the context associated with a request
type RequestCtx interface {
	Context

	// Request retrieves the original request
	Request() *http.Request

	// ResponseWriter retrieves the writer for generating response
	ResponseWriter() http.ResponseWriter

	// RequestID extract X-Request-ID from request
	// it can be empty for auth and hooks
	RequestID() string

	// Module returns the module which handles the request
	// this can be nil if the request is not for module
	Module() Module
}

// Scheduler schedules tasks
type Scheduler interface {
}

// SCMProvider is source control manager provider
type SCMProvider interface {
}

// Module is an extension component which adds feature to server
type Module interface {
	Name() string
	Init(InitCtx) error
}

// Annotatable represents anything that can be associated with
// certain runtime data
type Annotatable interface {
	Annotate(interface{})
	Annotation() interface{}
}

// Store abstraction
//
// Supported Read/Write patterns:
//   - Read only
//   - Read and write batch at last

// Tags represents a map of tags
type Tags interface {
	Get(name string) interface{}
	Set(name string, value interface{}) Tags
	Names() []string
	Reset(map[string]interface{}) Tags
}

// Meta represents opaque metadata associated with entity
type Meta interface {
	// ID retrieves the entity id
	ID() string
	// Tags retrieves custom tags
	Tags() Tags
}

// Entity represents the raw object
type Entity interface {
	Annotatable

	// Meta retrieve metadata
	Meta() Meta
	// Dirty indicates if there's any in-memory changes
	Dirty() bool
	// Decode retrieves the value
	Decode(out interface{}) error
	// Encode sets the value and returns itself
	Encode(in interface{}) Entity
	// Save updates the entity
	Save() UpdateOp
	// Delete deletes this single entity
	Delete() UpdateOp
}

// Query represents a query to the store
type Query interface {
	Annotatable

	// ID queries single item by ID
	ID(string) Query
	// Prefix sets prefix of the keys
	Prefix(string) Query
	// Filters retrieve filtering tags
	Filters() Tags
	// Limit set the maximum retrieved entities
	Limit(int) Query
	// PerPage limits the number of entities for each fetch
	PerPage(int) Query
	// Fetch executes the query and returns the resultset
	Fetch() (ResultSet, error)
	// Delete deletes all entities matching the query
	Delete() UpdateOp
	// String encodes the query into string
	String() string
}

// ResultSet represents query results
type ResultSet interface {
	io.Closer
	// Entities retrieves the queried entities
	Entities() []Entity
	// More returns the query for more entities
	// if returns nil, there's no more entities
	More() Query
	// Query retrieves the query which generates the resultset
	Query() Query
}

// UpdateOp represents one update operation
type UpdateOp interface {
	Annotatable

	// Entity retrieves the associated entity if the operation is for entity
	Entity() Entity
	// Query retrieves the query if the operation is for a query
	Query() Query
	// Result retrieves if the operation is successful or error
	// if the first bool value is false, the operation is not executed yet
	Result() (bool, error)
}

// Store represents queryable object store (K/V store)
type Store interface {
	io.Closer
	// Namespace opens a store associated with the namespace
	Namespace(string) Store
	// Select constructs a new query
	Select() Query
	// DecodeQuery decodes the string (encoded using Query.String) as a query
	DecodeQuery(encoded string) Query
	// NewEntity creates a new entity with ID
	NewEntity(id string) Entity
	// UpdateBatch perform multiple update operations
	UpdateBatch([]UpdateOp) error
}
