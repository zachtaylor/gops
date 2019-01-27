package main

import (
	"os"
	"regexp"
	"strings"

	"ztaylor.me/gops"
)

type Service struct {
	Method  string
	Handler func(HandlerReq) error
	Rpc     string
}

type HandlerReq struct {
	i    gops.In
	o    gops.Out
	Rpc  string
	Dir  string
	File string
}

// Routing regexes
var (
	_serviceRpcUpload  = regexp.MustCompile("(.*?)/git-upload-pack$")
	_serviceRpcReceive = regexp.MustCompile("(.*?)/git-receive-pack$")
	_getInfoRefs       = regexp.MustCompile("(.*?)/info/refs$")
	_getHead           = regexp.MustCompile("(.*?)/HEAD$")
	_getAlternates     = regexp.MustCompile("(.*?)/objects/info/alternates$")
	_getHttpAlternates = regexp.MustCompile("(.*?)/objects/info/http-alternates$")
	_getInfoPacks      = regexp.MustCompile("(.*?)/objects/info/packs$")
	_getInfoFile       = regexp.MustCompile("(.*?)/objects/info/[^/]*$")
	_getLooseObject    = regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$")
	_getPackFile       = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$")
	_getIdxFile        = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$")
)

func (g *GitHttp) services() map[*regexp.Regexp]Service {
	return map[*regexp.Regexp]Service{
		_serviceRpcUpload:  Service{"POST", g.serviceRpc, "upload-pack"},
		_serviceRpcReceive: Service{"POST", g.serviceRpc, "receive-pack"},
		_getInfoRefs:       Service{"GET", g.getInfoRefs, ""},
		_getHead:           Service{"GET", g.getTextFile, ""},
		_getAlternates:     Service{"GET", g.getTextFile, ""},
		_getHttpAlternates: Service{"GET", g.getTextFile, ""},
		_getInfoPacks:      Service{"GET", g.getInfoPacks, ""},
		_getInfoFile:       Service{"GET", g.getTextFile, ""},
		_getLooseObject:    Service{"GET", g.getLooseObject, ""},
		_getPackFile:       Service{"GET", g.getPackFile, ""},
		_getIdxFile:        Service{"GET", g.getIdxFile, ""},
	}
}

// getService return's the service corresponding to the
// current http.Request's URL
// as well as the name of the repo
func (g *GitHttp) getService(path string) (string, *Service) {
	for re, service := range g.services() {
		if m := re.FindStringSubmatch(path); m != nil {
			return m[1], &service
		}
	}

	// No match
	return "", nil
}

// Request handling function
func (g *GitHttp) requestHandler(i gops.In, o gops.Out) {
	// Get service for URL
	repo, service := g.getService(i.Path())

	// No url match
	if service == nil {
		renderNotFound(o)
		return
	}

	// Bad method
	if service.Method != i.Method() {
		renderMethodNotAllowed(i, o)
		return
	}

	// Rpc type
	rpc := service.Rpc

	// Get specific file
	file := strings.Replace(i.Path(), repo+"/", "", 1)

	// Resolve directory
	dir, err := g.getGitDir(repo)

	// Repo not found on disk
	if err != nil {
		renderNotFound(o)
		return
	}

	// Build request info for handler
	hr := HandlerReq{i, o, rpc, dir, file}

	// Call handler
	if err := service.Handler(hr); err != nil {
		if os.IsNotExist(err) {
			renderNotFound(o)
			return
		}
		switch err.(type) {
		case *ErrorNoAccess:
			renderNoAccess(o)
			return
		}
		Error(o, err.Error(), StatusInternalServerError)
	}
}