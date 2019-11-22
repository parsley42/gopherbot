// +build modular

package slack

import (
	"log"

	"github.com/lnxjedi/gopherbot/robot"
)

var slackspec = robot.PluginSpec{
	Name:    "slackutil",
	Handler: slackplugin,
}

// GetPlugins is the common exported symbol for loadable go plugins.
func GetPlugins() []robot.PluginSpec {
	return []robot.PluginSpec{
		slackspec,
	}
}

// GetInitializer is the common exported symbol for loadable connector modules.
func GetInitializer() (string, func(robot.Handler, *log.Logger) robot.Connector) {
	return "slack", Initialize
}
