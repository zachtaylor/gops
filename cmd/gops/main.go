package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"ztaylor.me/env"
	"ztaylor.me/http/mux"
	"ztaylor.me/log"
)

func main() {
	env := env.Global()
	path := env.Default("GOPS_PATH", "/srv/gops/")
	log.SetLevel(env.Default("LOG_LEVEL", "info"))
	log.WithFields(log.Fields{
		"path": path,
	}).Debug("gops: starting...")

	dir, err := ioutil.ReadDir(path)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("gops: failed to open GOPS_PATH")
		os.Exit(1)
	}

	server := mux.NewMux()

	for _, fi := range dir {
		if n := fi.Name(); len(n) < 3 || n[len(n)-3:] != ".so" {
			// continue
		} else if plugin, err := open(path + n); err != nil {
			log.WithFields(log.Fields{
				"File":  n,
				"Error": err.Error(),
			}).Error("gops: failed to open plugin")
		} else {
			server.Router(&adapter{plugin})
			log.WithFields(log.Fields{
				"File": n,
			}).Debug("gops: loaded plugin")
		}
	}

	log.Info("gops: starting")

	if port := env.Get("PORT"); len(port) > 1 {
		log.Error(http.ListenAndServe(":"+port, server))
	} else {
		go http.ListenAndServe(":80", server)
		log.Error(http.ListenAndServeTLS(":443", ".cert", ".key", server))
	}
}
