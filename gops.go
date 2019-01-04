// Package gops provides basic headers for use by GoPS Plugins, to reduce the
// size of compiled binaries versus net/http
package gops

// Plugin is the type imported by GoPS, the entry point for plugins
type Plugin interface {
	// Route returns whether this Plugin handles the input
	Route(In) bool
	// Handle responds to input, as in ServeHTTP
	Handle(In, Out)
}

// New creates a Plugin from a Router func and a Handler func
func New(r func(In) bool, h func(In, Out)) Plugin {
	return &t{r, h}
}

type t struct {
	r func(In) bool
	h func(In, Out)
}

func (t *t) Route(i In) bool {
	return t.r(i)
}

func (t *t) Handle(i In, o Out) {
	t.h(i, o)
}

// In is used by Plugin as a header for http.Request
type In interface {
	// Secure returns true if TLS != nil
	Secure() bool
	// Method returns Request method
	Method() string
	// Method returns Request host
	Host() string
	// Path returns Request URL path
	Path() string
	// Query returns the 1st value of given Request query param
	Query(string) string
	// Cookie returns the value of given Request cookie
	Cookie(string) string
}

// Out is used by Plugin as a header for http.ResponseWriter
type Out interface {
	// Header adds an entry to response headers
	Header(string, string)
	// StatusCode writes the status code
	StatusCode(int)
	// Write sends response data
	Write([]byte) (int, error)
}
