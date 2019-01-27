package main

import (
	"ztaylor.me/gops"
)

var Plugin = gops.New(
	router,
	server.Handle,
)

func router(i gops.In) bool {
	ua := i.Header("User-Agent")
	return len(ua) > 2 && ua[:3] == "git"
}

var server = GitHttp{
	ProjectRoot: "/srv/git",
	GitBinPath:  "/usr/bin/git",
	UploadPack:  true,
	ReceivePack: false,
}

func main() {
}

func getHeader(o gops.Out, k string) string {
	if all := o.Headers()[k]; len(all) > 0 {
		return all[0]
	}
	return ""
}
