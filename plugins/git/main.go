package main

import (
	"ztaylor.me/gops"
)

var Plugin = gops.New(
	gops.RouterFunc(router),
	server,
)

func router(i gops.In) bool {
	ua := i.Header("User-Agent")
	return len(ua) > 2 && ua[:3] == "git"
}

var server = &GitHttp{
	ProjectRoot: "/srv/git",
	GitBinPath:  "/usr/bin/git",
	UploadPack:  true,
	ReceivePack: false,
}

func main() {
}
