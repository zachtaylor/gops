# GoPS (Golang Plugin Server)

Provides a basic server, using Go 1.8 plugin system

# Package gops

```
import "ztaylor.me/gops"
```

Provides basic HTTP headers, for use by plugins, to interface with `net/http`

# Command `gops`

```
... $ go install ztaylor.me/gops/cmd/gops
```

Executable which serves plugins

## Options

```
GOPS_PATH   path to use in place of /srv/gops/

LOG_LEVEL   one of ["debug","info","warn","error"]

PORT        GoPS starts using only this port
```

# Plugins

Plugins are go executables (read: package `main`) built with the arg `-buildmode=plugin`

Plugins must expose a variable named `Plugin` of type `gops.Plugin`
