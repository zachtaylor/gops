// Package gops provides headers for GoPS plugins
package gops

import "io"

// In is an interface for http.Request
type In interface {
	// Secure returns Request TLS != nil
	Secure() bool
	// Method returns Request method
	Method() string
	// Proto returns Request proto
	Proto() string
	// Method returns Request host
	Host() string
	// Path returns Request URL path
	Path() string
	// Header returns Request header 1st value
	Header(string) string
	// RawQuery returns Request raw query
	RawQuery() string
	// Query returns Request query 1st value
	Query(string) string
	// FormValue returns Request form value
	FormValue(string) string
	// Cookie returns Request cookie value
	Cookie(string) string
	// Body returns Request body
	Body() io.ReadCloser
}

// Out is an interface for http.ResponseWriter
type Out interface {
	io.Writer
	// Headers returns ResponseWriter headers
	Headers() map[string][]string
	// Header adds to ResponseWriter headers
	Header(string, string)
	// StatusCode writes the ResponseWriter status code
	StatusCode(int)
}

// Router is an interface for request matching
type Router interface {
	// Route tests an input
	Route(In) bool
}

// Handler is an interface for request handling
type Handler interface {
	// Handle responds to input
	Handle(In, Out)
}

// Plugin is an interface for import by cmd/gops
type Plugin interface {
	Router
	Handler
}

// New creates a Plugin from Router and Handler
func New(r Router, h Handler) Plugin {
	return &plugin{r, h}
}

type plugin struct {
	Router  Router
	Handler Handler
}

func (plugin *plugin) Route(i In) bool {
	return plugin.Router.Route(i)
}

func (plugin *plugin) Handle(i In, o Out) {
	plugin.Handler.Handle(i, o)
}

// Mux is a slice of Plugin that also satisfies Plugin
type Mux []Plugin

// Route satisfies Router by calling each member Plugin
func (mux Mux) Route(i In) bool {
	for _, router := range mux {
		if !router.Route(i) {
			return false
		}
	}
	return true
}

// Handle satisfies Handler by calling each member Plugin
func (mux Mux) Handle(i In, o Out) {
	for _, router := range mux {
		if router.Route(i) {
			router.Handle(i, o)
			return
		}
	}
}
