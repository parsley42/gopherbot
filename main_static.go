// +build !modular

package main

// blank imports used in static builds
import (
	_ "github.com/lnxjedi/gopherbot/connectors/slack"

	_ "github.com/lnxjedi/gopherbot/goplugins/knock"
)
