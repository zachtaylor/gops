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

type in struct {
	Request *http.Request
}

func (i in) Secure() bool {
	return i.Request.TLS != nil
}

func (i in) Method() string {
	return i.Request.Method
}

func (i in) Proto() string {
	return i.Request.Proto
}

func (i in) Host() string {
	return i.Request.Host
}

func (i in) Path() string {
	return i.Request.URL.Path
}

func (i in) Header(k string) string {
	return i.Request.Header.Get(k)
}

func (i in) RawQuery() string {
	return i.Request.URL.RawQuery
}

func (i in) Query(k string) string {
	if q := i.Request.URL.Query()[k]; q != nil && len(q) > 0 {
		return q[0]
	}
	return ""
}

func (i in) FormValue(k string) string {
	return i.Request.FormValue(k)
}

func (i in) Cookie(k string) string {
	if c, err := i.Request.Cookie(k); err != nil {
		return ""
	} else {
		return c.String()
	}
}

func (i in) Body() gops.ReadCloser {
	return i.Request.Body
}

type out struct {
	ResponseWriter http.ResponseWriter
}

func (o out) Headers() map[string][]string {
	return o.ResponseWriter.Header()
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

type adapter struct {
	gops.Plugin
}

func (a *adapter) Route(r *http.Request) bool {
	return a.Plugin.Route(in{r})
}

func (a *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Plugin.Handle(in{r}, out{w})
}
