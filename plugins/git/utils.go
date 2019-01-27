package main

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"ztaylor.me/gops"
)

// requestReader returns an io.ReadCloser
// that will decode data if needed, depending on the
// "content-encoding" header
func requestReader(i gops.In) (io.ReadCloser, error) {
	switch i.Header("content-encoding") {
	case "gzip":
		return gzip.NewReader(i.Body())
	case "deflate":
		return flate.NewReader(i.Body()), nil
	}

	// If no encoding, use raw body
	return i.Body(), nil
}

// HTTP parsing utility functions

func getServiceType(i gops.In) string {
	service_type := i.FormValue("service")

	if s := strings.HasPrefix(service_type, "git-"); !s {
		return ""
	}

	return strings.Replace(service_type, "git-", "", 1)
}

// HTTP error response handling functions

func renderMethodNotAllowed(i gops.In, o gops.Out) {
	if i.Proto() == "HTTP/1.1" {
		o.StatusCode(StatusMethodNotAllowed)
		o.Write([]byte("Method Not Allowed"))
	} else {
		o.StatusCode(StatusBadRequest)
		o.Write([]byte("Bad Request"))
	}
}

func renderNotFound(o gops.Out) {
	o.StatusCode(StatusNotFound)
	o.Write([]byte("Not Found"))
}

func renderNoAccess(o gops.Out) {
	o.StatusCode(StatusForbidden)
	o.Write([]byte("Forbidden"))
}

// Packet-line handling function

func packetFlush() []byte {
	return []byte("0000")
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)

	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}

	return []byte(s + str)
}

// Header writing functions

func hdrNocache(o gops.Out) {
	o.Header("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	o.Header("Pragma", "no-cache")
	o.Header("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func hdrCacheForever(o gops.Out) {
	now := time.Now().Unix()
	expires := now + 31536000
	o.Header("Date", fmt.Sprintf("%d", now))
	o.Header("Expires", fmt.Sprintf("%d", expires))
	o.Header("Cache-Control", "public, max-age=31536000")
}
