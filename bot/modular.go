// +build modular

package bot

import (
	"plugin"

	"github.com/lnxjedi/gopherbot/robot"
)

// Load pluggable modules and call "GetPlugins", "GetConnectors", etc., then
// register them.
func loadModules() {
	for _, m := range botCfg.loadableModules {
		loadModule(m.Name, m.Path)
	}
}

// loadModule loads a module and registers it's contents
func loadModule(name, path string) {
	lp, err := getObjectPath(path)
	if err != nil {
		Log(robot.Warn, "unable to locate loadable module '%s' from path '%s'", name, path)
		return
	}
	if k, err := plugin.Open(lp); err == nil {
		if gp, err := k.Lookup("GetPlugins"); err == nil {
			gf := gp.(func() []robot.PluginSpec)
			pl := gf()
			for _, pspec := range pl {
				Log(robot.Info, "registered plugin '%s' from loadable module '%s'", pspec.Name, path)
				RegisterPlugin(pspec.Name, pspec.Handler)
			}
		} else {
			Log(robot.Debug, "symbol 'GetPlugins' not found in loadable module '%s': %v", path, err)
		}
	} else {
		Log(robot.Error, "loading module '%s': %v", lp, err)
	}
}
