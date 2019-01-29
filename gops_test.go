package gops_test

import "io"

type input struct {
	secure    bool
	method    string
	proto     string
	host      string
	path      string
	header    map[string]string
	rawquery  string
	query     map[string]string
	formvalue map[string]string
	cookie    map[string]string
	body      io.ReadCloser
}

func NewInput() *input {
	return &input{
		header:    make(map[string]string),
		query:     make(map[string]string),
		formvalue: make(map[string]string),
		cookie:    make(map[string]string),
	}
}

func (in *input) Secure() bool {
	return in.secure
}

func (in *input) Method() string {
	return in.method
}

func (in *input) Proto() string {
	return in.proto
}

func (in *input) Host() string {
	return in.host
}

func (in *input) Path() string {
	return in.path
}

func (in *input) Header(k string) string {
	return in.header[k]
}

func (in *input) RawQuery() string {
	return in.rawquery
}

func (in *input) Query(k string) string {
	return in.query[k]
}

func (in *input) FormValue(k string) string {
	return in.formvalue[k]
}

func (in *input) Cookie(k string) string {
	return in.cookie[k]
}

func (in *input) Body() io.ReadCloser {
	return in.body
}
