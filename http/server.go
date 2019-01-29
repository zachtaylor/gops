// Original copyright 2011 The Go Authors
package http

import (
	"fmt"
	"strings"

	"ztaylor.me/gops"
)

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&#34;",
	"'", "&#39;",
)

func Error(o gops.Out, error string, code int) {
	o.Header("Content-Type", "text/plain; charset=utf-8")
	o.Header("X-Content-Type-Options", "nosniff")
	o.StatusCode(code)
	fmt.Fprintln(o, error)
}
