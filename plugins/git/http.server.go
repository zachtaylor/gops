// Original copyright 2011 The Go Authors
package main

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
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

func Error(o gops.Out, error string, code int) {
	o.Header("Content-Type", "text/plain; charset=utf-8")
	o.Header("X-Content-Type-Options", "nosniff")
	o.StatusCode(code)
	fmt.Fprintln(o, error)
}
