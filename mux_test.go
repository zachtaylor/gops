package gops_test

import (
	"testing"

	"ztaylor.me/gops"
)

func TestRouterPath(t *testing.T) {
	router := gops.RouterPath("/hello/")

	in := NewInput()

	in.path = "/hello/"

	if !router.Route(in) {
		t.Fail()
	}

	in.path = "/hello/world"

	if !router.Route(in) {
		t.Fail()
	}

	in.path = "/hello"

	if router.Route(in) {
		t.Fail()
	}
}
