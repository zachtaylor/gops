// Original copyright 2011 The Go Authors
package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ztaylor.me/gops"
)

type Dir string

func mapDirOpenError(originalErr error, name string) error {
	if os.IsNotExist(originalErr) || os.IsPermission(originalErr) {
		return originalErr
	}

	parts := strings.Split(name, string(filepath.Separator))
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		fi, err := os.Stat(strings.Join(parts[:i+1], string(filepath.Separator)))
		if err != nil {
			return originalErr
		}
		if !fi.IsDir() {
			return os.ErrNotExist
		}
	}
	return originalErr
}

func (d Dir) Open(name string) (File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}
	dir := string(d)
	if dir == "" {
		dir = "."
	}
	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	f, err := os.Open(fullName)
	if err != nil {
		return nil, mapDirOpenError(err, fullName)
	}
	return f, nil
}

type FileSystem interface {
	Open(name string) (File, error)
}

type File interface {
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
}

func dirList(i gops.In, o gops.Out, f File) {
	dirs, err := f.Readdir(-1)
	if err != nil {
		Error(o, "Error reading directory", StatusInternalServerError)
		return
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	o.Header("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(o, "<pre>\n")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		url := url.URL{Path: name}
		fmt.Fprintf(o, "<a href=\"%s\">%s</a>\n", url.String(), htmlReplacer.Replace(name))
	}
	fmt.Fprintf(o, "</pre>\n")
}

func ServeContent(i gops.In, o gops.Out, name string, modtime time.Time, content io.ReadSeeker) {
	sizeFunc := func() (int64, error) {
		size, err := content.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, errSeeker
		}
		_, err = content.Seek(0, io.SeekStart)
		if err != nil {
			return 0, errSeeker
		}
		return size, nil
	}
	serveContent(i, o, name, modtime, sizeFunc, content)
}

var errSeeker = errors.New("seeker can't seek")

var errNoOverlap = errors.New("invalid range: failed to overlap")

func serveContent(i gops.In, o gops.Out, name string, modtime time.Time, sizeFunc func() (int64, error), content io.ReadSeeker) {
	setLastModified(o, modtime)
	done, rangeReq := checkPreconditions(i, o, modtime)
	if done {
		return
	}

	code := StatusOK

	ctypes, haveType := o.Headers()["Content-Type"]
	var ctype string
	if !haveType {
		ctype = mime.TypeByExtension(filepath.Ext(name))
		if ctype == "" {
			var buf [sniffLen]byte
			n, _ := io.ReadFull(content, buf[:])
			ctype = DetectContentType(buf[:n])
			_, err := content.Seek(0, io.SeekStart)
			if err != nil {
				Error(o, "seeker can't seek", StatusInternalServerError)
				return
			}
		}
		o.Header("Content-Type", ctype)
	} else if len(ctypes) > 0 {
		ctype = ctypes[0]
	}

	size, err := sizeFunc()
	if err != nil {
		Error(o, err.Error(), StatusInternalServerError)
		return
	}

	sendSize := size
	var sendContent io.Reader = content
	if size >= 0 {
		ranges, err := parseRange(rangeReq, size)
		if err != nil {
			if err == errNoOverlap {
				o.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
			}
			Error(o, err.Error(), StatusRequestedRangeNotSatisfiable)
			return
		}
		if sumRangesSize(ranges) > size {
			ranges = nil
		}
		switch {
		case len(ranges) == 1:
			ra := ranges[0]
			if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
				Error(o, err.Error(), StatusRequestedRangeNotSatisfiable)
				return
			}
			sendSize = ra.length
			code = StatusPartialContent
			o.Header("Content-Range", ra.contentRange(size))
		case len(ranges) > 1:
			sendSize = rangesMIMESize(ranges, ctype, size)
			code = StatusPartialContent

			pr, pw := io.Pipe()
			mw := multipart.NewWriter(pw)
			o.Header("Content-Type", "multipart/byteranges; boundary="+mw.Boundary())
			sendContent = pr
			defer pr.Close()
			go func() {
				for _, ra := range ranges {
					part, err := mw.CreatePart(ra.mimeHeader(ctype, size))
					if err != nil {
						pw.CloseWithError(err)
						return
					}
					if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
						pw.CloseWithError(err)
						return
					}
					if _, err := io.CopyN(part, content, ra.length); err != nil {
						pw.CloseWithError(err)
						return
					}
				}
				mw.Close()
				pw.Close()
			}()
		}

		o.Header("Accept-Ranges", "bytes")
		if getHeader(o, "Content-Encoding") == "" {
			o.Header("Content-Length", strconv.FormatInt(sendSize, 10))
		}
	}

	o.StatusCode(code)

	if i.Method() != "HEAD" {
		io.CopyN(o, sendContent, sendSize)
	}
}

func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return s[:i+1], s[i+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}

func etagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

func etagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}

type condResult int

const (
	condNone condResult = iota
	condTrue
	condFalse
)

func checkIfMatch(i gops.In, o gops.Out) condResult {
	im := i.Header("If-Match")
	if im == "" {
		return condNone
	}
	for {
		im = textproto.TrimString(im)
		if len(im) == 0 {
			break
		}
		if im[0] == ',' {
			im = im[1:]
			continue
		}
		if im[0] == '*' {
			return condTrue
		}
		etag, remain := scanETag(im)
		if etag == "" {
			break
		}
		if etagStrongMatch(etag, getHeader(o, "Etag")) {
			return condTrue
		}
		im = remain
	}

	return condFalse
}

func checkIfUnmodifiedSince(i gops.In, modtime time.Time) condResult {
	ius := i.Header("If-Unmodified-Since")
	if ius == "" || isZeroTime(modtime) {
		return condNone
	}
	if t, err := ParseTime(ius); err == nil {
		if modtime.Before(t.Add(1 * time.Second)) {
			return condTrue
		}
		return condFalse
	}
	return condNone
}

func checkIfNoneMatch(i gops.In, o gops.Out) condResult {
	inm := i.Header("If-None-Match")
	if inm == "" {
		return condNone
	}
	buf := inm
	for {
		buf = textproto.TrimString(buf)
		if len(buf) == 0 {
			break
		}
		if buf[0] == ',' {
			buf = buf[1:]
		}
		if buf[0] == '*' {
			return condFalse
		}
		etag, remain := scanETag(buf)
		if etag == "" {
			break
		}
		if etagWeakMatch(etag, getHeader(o, "Etag")) {
			return condFalse
		}
		buf = remain
	}
	return condTrue
}

func checkIfModifiedSince(i gops.In, modtime time.Time) condResult {
	if i.Method() != "GET" && i.Method() != "HEAD" {
		return condNone
	}
	ims := i.Header("If-Modified-Since")
	if ims == "" || isZeroTime(modtime) {
		return condNone
	}
	t, err := ParseTime(ims)
	if err != nil {
		return condNone
	}
	if modtime.Before(t.Add(1 * time.Second)) {
		return condFalse
	}
	return condTrue
}

func checkIfRange(i gops.In, o gops.Out, modtime time.Time) condResult {
	if i.Method() != "GET" && i.Method() != "HEAD" {
		return condNone
	}
	ir := i.Header("If-Range")
	if ir == "" {
		return condNone
	}
	etag, _ := scanETag(ir)
	if etag != "" {
		if etagStrongMatch(etag, getHeader(o, "Etag")) {
			return condTrue
		} else {
			return condFalse
		}
	}
	if modtime.IsZero() {
		return condFalse
	}
	t, err := ParseTime(ir)
	if err != nil {
		return condFalse
	}
	if t.Unix() == modtime.Unix() {
		return condTrue
	}
	return condFalse
}

var unixEpochTime = time.Unix(0, 0)

func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

func setLastModified(o gops.Out, modtime time.Time) {
	if !isZeroTime(modtime) {
		o.Header("Last-Modified", modtime.UTC().Format(TimeFormat))
	}
}

func writeNotModified(o gops.Out) {
	h := o.Headers()
	delete(h, "Content-Type")
	delete(h, "Content-Length")
	if getHeader(o, "Etag") != "" {
		delete(h, "Last-Modified")
	}
	o.StatusCode(StatusNotModified)
}

func checkPreconditions(i gops.In, o gops.Out, modtime time.Time) (done bool, rangeHeader string) {
	ch := checkIfMatch(i, o)
	if ch == condNone {
		ch = checkIfUnmodifiedSince(i, modtime)
	}
	if ch == condFalse {
		o.StatusCode(StatusPreconditionFailed)
		return true, ""
	}
	switch checkIfNoneMatch(i, o) {
	case condFalse:
		if i.Method() == "GET" || i.Method() == "HEAD" {
			writeNotModified(o)
			return true, ""
		} else {
			o.StatusCode(StatusPreconditionFailed)
			return true, ""
		}
	case condNone:
		if checkIfModifiedSince(i, modtime) == condFalse {
			writeNotModified(o)
			return true, ""
		}
	}

	rangeHeader = i.Header("Range")
	if rangeHeader != "" && checkIfRange(i, o, modtime) == condFalse {
		rangeHeader = ""
	}
	return false, rangeHeader
}

func serveFile(i gops.In, o gops.Out, fs FileSystem, name string, redirect bool) {
	const indexPage = "/index.html"

	if strings.HasSuffix(i.Path(), indexPage) {
		localRedirect(i, o, "./")
		return
	}

	f, err := fs.Open(name)
	if err != nil {
		msg, code := toHTTPError(err)
		Error(o, msg, code)
		return
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		msg, code := toHTTPError(err)
		Error(o, msg, code)
		return
	}

	if redirect {
		url := i.Path()
		if d.IsDir() {
			if url[len(url)-1] != '/' {
				localRedirect(i, o, path.Base(url)+"/")
				return
			}
		} else {
			if url[len(url)-1] == '/' {
				localRedirect(i, o, "../"+path.Base(url))
				return
			}
		}
	}

	if d.IsDir() {
		url := i.Path()
		if url[len(url)-1] != '/' {
			localRedirect(i, o, path.Base(url)+"/")
			return
		}
	}

	if d.IsDir() {
		index := strings.TrimSuffix(name, "/") + indexPage
		ff, err := fs.Open(index)
		if err == nil {
			defer ff.Close()
			dd, err := ff.Stat()
			if err == nil {
				name = index
				d = dd
				f = ff
			}
		}
	}

	if d.IsDir() {
		if checkIfModifiedSince(i, d.ModTime()) == condFalse {
			writeNotModified(o)
			return
		}
		o.Header("Last-Modified", d.ModTime().UTC().Format(TimeFormat))
		dirList(i, o, f)
		return
	}

	sizeFunc := func() (int64, error) { return d.Size(), nil }
	serveContent(i, o, d.Name(), d.ModTime(), sizeFunc, f)
}

func toHTTPError(err error) (msg string, httpStatus int) {
	if os.IsNotExist(err) {
		return "404 page not found", StatusNotFound
	}
	if os.IsPermission(err) {
		return "403 Forbidden", StatusForbidden
	}
	return "500 Internal Server Error", StatusInternalServerError
}

func localRedirect(i gops.In, o gops.Out, newPath string) {
	if q := i.RawQuery(); q != "" {
		newPath += "?" + q
	}
	o.Header("Location", newPath)
	o.StatusCode(StatusMovedPermanently)
}

func ServeFile(i gops.In, o gops.Out, name string) {
	if containsDotDot(i.Path()) {
		Error(o, "invalid URL path", StatusBadRequest)
		return
	}
	dir, file := filepath.Split(name)
	serveFile(i, o, Dir(dir), file, false)
}

func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

// type fileHandler struct {
// 	root FileSystem
// }

// func FileServer(root FileSystem) Handler {
// 	return &fileHandler{root}
// }

// func (f *fileHandler) Handle(i gops.In, o gops.Out) {
// 	upath := i.Path()
// 	if !strings.HasPrefix(upath, "/") {
// 		upath = "/" + upath
// 		r.URL.Path = upath
// 	}
// 	serveFile(i, o, f.root, path.Clean(upath), true)
// }

type httpRange struct {
	start, length int64
}

func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

func (r httpRange) mimeHeader(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.contentRange(size)},
		"Content-Type":  {contentType},
	}
}

func parseRange(s string, size int64) ([]httpRange, error) {
	if s == "" {
		return nil, nil
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, errors.New("invalid range")
	}
	var ranges []httpRange
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = strings.TrimSpace(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, errors.New("invalid range")
		}
		start, end := strings.TrimSpace(ra[:i]), strings.TrimSpace(ra[i+1:])
		var r httpRange
		if start == "" {
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return nil, errors.New("invalid range")
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, errors.New("invalid range")
			}
			if i >= size {
				noOverlap = true
				continue
			}
			r.start = i
			if end == "" {
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, errors.New("invalid range")
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		return nil, errNoOverlap
	}
	return ranges, nil
}

type countingWriter int64

func (w *countingWriter) Write(p []byte) (n int, err error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

func rangesMIMESize(ranges []httpRange, contentType string, contentSize int64) (encSize int64) {
	var w countingWriter
	mw := multipart.NewWriter(&w)
	for _, ra := range ranges {
		mw.CreatePart(ra.mimeHeader(contentType, contentSize))
		encSize += ra.length
	}
	mw.Close()
	encSize += int64(w)
	return
}

func sumRangesSize(ranges []httpRange) (size int64) {
	for _, ra := range ranges {
		size += ra.length
	}
	return
}
