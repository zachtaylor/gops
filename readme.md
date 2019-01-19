# GoPS (Golang Plugin Server)

Provides a basic server, using Go 1.8 plugin system

# Package `gops`

```
import "ztaylor.me/gops"
```

Provides basic IO pattern, to interface with `net/http`

# Command `gops`

```
... $ go install ztaylor.me/gops/cmd/gops
```

Executable server host that requires plugins

## Options

```
GOPS_PATH     path to load plugins from (default: /srv/gops/)

LOG_LEVEL     one of ["debug","info","warn","error"] (default: info)

PORT          GoPS starts using only this single port (default: 80 and 443)
```

# Plugins

Plugins are go `main` packages built with `-buildmode=plugin`

Plugins must expose a variable named `Plugin` of type `gops.Plugin` to be imported by GoPS
