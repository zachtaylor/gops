package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"ztaylor.me/gops"
	"ztaylor.me/gops/http"
)

type GitHttp struct {
	// Root directory to serve repos from
	ProjectRoot string

	// Path to git binary
	GitBinPath string

	// Access rules
	UploadPack  bool
	ReceivePack bool

	// Event handling functions
	EventHandler func(ev Event)
}

// Implement the http.Handler interface
func (g *GitHttp) Handle(i gops.In, o gops.Out) {
	g.requestHandler(i, o)
	return
}

// Shorthand constructor for most common scenario
func New(root string) *GitHttp {
	return &GitHttp{
		ProjectRoot: root,
		GitBinPath:  "/usr/bin/git",
		UploadPack:  true,
		ReceivePack: true,
	}
}

// Build root directory if doesn't exist
func (g *GitHttp) Init() (*GitHttp, error) {
	if err := os.MkdirAll(g.ProjectRoot, os.ModePerm); err != nil {
		return nil, err
	}
	return g, nil
}

// Publish event if EventHandler is set
func (g *GitHttp) event(e Event) {
	if g.EventHandler != nil {
		g.EventHandler(e)
	} else {
		fmt.Printf("EVENT: %q\n", e)
	}
}

// Actual command handling functions

func (g *GitHttp) serviceRpc(hr HandlerReq) error {
	i, o, rpc, dir := hr.i, hr.o, hr.Rpc, hr.Dir

	access, err := g.hasAccess(i, dir, rpc, true)
	if err != nil {
		return err
	}

	if access == false {
		return &ErrorNoAccess{hr.Dir}
	}

	// Reader that decompresses if necessary
	reader, err := requestReader(i)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Reader that scans for events
	rpcReader := &RpcReader{
		Reader: reader,
		Rpc:    rpc,
	}

	// Set content type
	o.Header("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))

	args := []string{rpc, "--stateless-rpc", "."}
	cmd := exec.Command(g.GitBinPath, args...)
	cmd.Dir = dir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Scan's git command's output for errors
	gitReader := &GitReader{
		Reader: stdout,
	}

	// Copy input to git binary
	io.Copy(stdin, rpcReader)
	stdin.Close()

	// Write git binary's output to http response
	io.Copy(o, gitReader)

	// Wait till command has completed
	mainError := cmd.Wait()

	if mainError == nil {
		mainError = gitReader.GitError
	}

	// Fire events
	for _, e := range rpcReader.Events {
		// Set directory to current repo
		e.Dir = dir
		e.I = hr.i
		e.Error = mainError

		// Fire event
		g.event(e)
	}

	// Because a response was already written,
	// the header cannot be changed
	return nil
}

func (g *GitHttp) getInfoRefs(hr HandlerReq) error {
	i, o, dir := hr.i, hr.o, hr.Dir
	service_name := getServiceType(i)
	access, err := g.hasAccess(i, dir, service_name, false)
	if err != nil {
		return err
	}

	if !access {
		g.updateServerInfo(dir)
		hdrNocache(o)
		return sendFile("text/plain; charset=utf-8", hr)
	}

	args := []string{service_name, "--stateless-rpc", "--advertise-refs", "."}
	refs, err := g.gitCommand(dir, args...)
	if err != nil {
		return err
	}

	hdrNocache(o)
	o.Header("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service_name))
	o.StatusCode(http.StatusOK)
	o.Write(packetWrite("# service=git-" + service_name + "\n"))
	o.Write(packetFlush())
	o.Write(refs)

	return nil
}

func (g *GitHttp) getInfoPacks(hr HandlerReq) error {
	hdrCacheForever(hr.o)
	return sendFile("text/plain; charset=utf-8", hr)
}

func (g *GitHttp) getLooseObject(hr HandlerReq) error {
	hdrCacheForever(hr.o)
	return sendFile("application/x-git-loose-object", hr)
}

func (g *GitHttp) getPackFile(hr HandlerReq) error {
	hdrCacheForever(hr.o)
	return sendFile("application/x-git-packed-objects", hr)
}

func (g *GitHttp) getIdxFile(hr HandlerReq) error {
	hdrCacheForever(hr.o)
	return sendFile("application/x-git-packed-objects-toc", hr)
}

func (g *GitHttp) getTextFile(hr HandlerReq) error {
	hdrNocache(hr.o)
	return sendFile("text/plain", hr)
}

// Logic helping functions

func sendFile(content_type string, hr HandlerReq) error {
	i, o := hr.i, hr.o
	req_file := path.Join(hr.Dir, hr.File)

	f, err := os.Stat(req_file)
	if err != nil {
		return err
	}

	o.Header("Content-Type", content_type)
	o.Header("Content-Length", fmt.Sprintf("%d", f.Size()))
	o.Header("Last-Modified", f.ModTime().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	http.ServeFile(i, o, req_file)

	return nil
}

func (g *GitHttp) getGitDir(file_path string) (string, error) {
	root := g.ProjectRoot

	if root == "" {
		cwd, err := os.Getwd()

		if err != nil {
			return "", err
		}

		root = cwd
	}

	f := path.Join(root, file_path)
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return "", err
	}

	return f, nil
}

func (g *GitHttp) hasAccess(i gops.In, dir string, rpc string, check_content_type bool) (bool, error) {
	if check_content_type {
		if i.Header("Content-Type") != fmt.Sprintf("application/x-git-%s-request", rpc) {
			return false, nil
		}
	}

	if !(rpc == "upload-pack" || rpc == "receive-pack") {
		return false, nil
	}
	if rpc == "receive-pack" {
		return g.ReceivePack, nil
	}
	if rpc == "upload-pack" {
		return g.UploadPack, nil
	}

	return g.getConfigSetting(rpc, dir)
}

func (g *GitHttp) getConfigSetting(service_name string, dir string) (bool, error) {
	service_name = strings.Replace(service_name, "-", "", -1)
	setting, err := g.getGitConfig("http."+service_name, dir)
	if err != nil {
		return false, nil
	}

	if service_name == "uploadpack" {
		return setting != "false", nil
	}

	return setting == "true", nil
}

func (g *GitHttp) getGitConfig(config_name string, dir string) (string, error) {
	args := []string{"config", config_name}
	out, err := g.gitCommand(dir, args...)
	if err != nil {
		return "", err
	}
	return string(out)[0 : len(out)-1], nil
}

func (g *GitHttp) updateServerInfo(dir string) ([]byte, error) {
	args := []string{"update-server-info"}
	return g.gitCommand(dir, args...)
}

func (g *GitHttp) gitCommand(dir string, args ...string) ([]byte, error) {
	command := exec.Command(g.GitBinPath, args...)
	command.Dir = dir

	return command.Output()
}
