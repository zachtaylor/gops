package main

import (
	"errors"
	"net/http"
	"plugin"

	"ztaylor.me/gops"
)

var errPluginReadFailed = errors.New(`failed to read plugin`)
var errPluginMissing = errors.New(`plugin missing`)
var errPluginType = errors.New(`plugin failed type conversion`)

func open(path string) (gops.Plugin, error) {
	if so, err := plugin.Open(path); err != nil {
		return nil, errPluginReadFailed
	} else if sr, err := so.Lookup("Plugin"); err != nil {
		return nil, errPluginMissing
	} else if plugin, ok := sr.(*gops.Plugin); !ok {
		return nil, errPluginType
	} else {
		return *plugin, nil
	}
}

type adapter struct {
	gops.Plugin
}

type in struct {
	*http.Request
}

func (i in) Secure() bool {
	return i.TLS != nil
}

func (i in) Method() string {
	return i.Method()
}

func (i in) Host() string {
	return i.Host()
}

func (i in) Path() string {
	return i.URL.Path
}

func (i in) Query(k string) string {
	if q := i.URL.Query()[k]; q != nil && len(q) > 0 {
		return q[0]
	}
	return ""
}

func (i in) Cookie(k string) string {
	return i.Cookie(k)
}

type out struct {
	http.ResponseWriter
}

func (o out) Header(k, v string) {
	o.ResponseWriter.Header().Add(k, v)
}

func (o out) StatusCode(c int) {
	o.ResponseWriter.WriteHeader(c)
}

func (o out) Write(data []byte) (int, error) {
	return o.ResponseWriter.Write(data)
}

func (a *adapter) Route(r *http.Request) bool {
	return a.Plugin.Route(in{r})
}

func (a *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Plugin.Handle(in{r}, out{w})
}
