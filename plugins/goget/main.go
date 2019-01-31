package main

import (
	"fmt"

	"ztaylor.me/gops"
)

var Plugin = gops.New(
	gops.RouterFunc(router),
	gops.HandlerFunc(handler),
)

func router(i gops.In) bool {
	ua := i.Header("User-Agent")
	return len(ua) > 13 && ua[:14] == "Go-http-client"
}

func handler(i gops.In, o gops.Out) {
	pkg := i.Path()
	o.Header("Content-Type", "text/html; charset=utf-8")
	o.StatusCode(500)
	fmt.Fprintf(o, text, pkg, pkg)
}

func main() {
}

const text = `<html>
	<meta name="go-import" content="ztaylor.me%s git https://ztaylor.me%s">
</html>
`
