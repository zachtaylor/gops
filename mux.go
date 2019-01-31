package gops

// HandlerFunc casts Handler from a basic func
type HandlerFunc func(In, Out)

// Handle satisfies Handler by calling the func
func (f HandlerFunc) Handle(i In, o Out) {
	f(i, o)
}

// RouterDomain creates a Router for Request domain from given string
type RouterDomain string

// Route satisfies Router by matching the Request host
func (router RouterDomain) Route(i In) bool {
	return string(router) == i.Host()
}

// RouterFunc casts Router from a basic func
type RouterFunc func(In) bool

// Route satisfies Router by calling the func
func (router RouterFunc) Route(i In) bool {
	return router(i)
}

var (
	// RouterHTTP is a Router that returns if Request TLS is nil
	RouterHTTP = RouterFunc(func(i In) bool {
		return !i.Secure()
	})

	// RouterHTTPS is a Router that returns if Request TLS is non-nil
	RouterHTTPS = RouterFunc(func(i In) bool {
		return i.Secure()
	})
)

// RouterPath is a Router for Request path starting with given string
type RouterPath string

// Route satisfies Router by matching the given prefix
func (router RouterPath) Route(i In) bool {
	if len(i.Path()) < len(router) {
		return false
	}
	return string(router) == i.Path()[:len(router)]
}

// RouterSet creates a Router from any number of Routers
func RouterSet(routers ...Router) Router {
	return RouterFunc(func(i In) bool {
		for _, r := range routers {
			if !r.Route(i) {
				return false
			}
		}
		return true
	})
}

type routerMethod string

func (router routerMethod) Route(i In) bool {
	return string(router) == i.Method()
}

var (
	// RouterCONNECT is a Router that returns if Request method is CONNECT
	RouterCONNECT = routerMethod("CONNECT")

	// RouterDELETE is a Router that returns if Request method is DELETE
	RouterDELETE = routerMethod("DELETE")

	// RouterGET is a Router that returns if Request method is GET
	RouterGET = routerMethod("GET")

	// RouterHEAD is a Router that returns if Request method is HEAD
	RouterHEAD = routerMethod("HEAD")

	// RouterOPTIONS is a Router that returns if Request method is OPTIONS
	RouterOPTIONS = routerMethod("OPTIONS")

	// RouterPOST is a Router that returns if Request method is POST
	RouterPOST = routerMethod("POST")

	// RouterPUT is a Router that returns if Request method is PUT
	RouterPUT = routerMethod("PUT")

	// RouterTRACE is a Router that returns if Request method is TRACE
	RouterTRACE = routerMethod("TRACE")
)
