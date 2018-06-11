package bot

import (
	"reflect"

	"github.com/ghodss/yaml"
)

// loadTaskConfig() loads the configuration for all the jobs/plugins from
// /jobs/<jobname>.yaml or /plugins/<pluginname>.yaml, assigns a taskID, and
// stores the resulting array in b.tasks. Bad tasks are skipped and logged.
// Task configuration is initially loaded into temporary data structures,
// then stored in the bot package under the global bot lock.
func (r *Robot) loadTaskConfig() {
	taskIndexByID := make(map[string]int)
	taskIndexByName := make(map[string]int)
	tlist := make([]interface{}, 0, 14)

	// Copy some data from the bot under read lock, including external plugins
	robot.RLock()
	defaultAllowDirect := robot.defaultAllowDirect
	// copy the list of default channels (for plugins only)
	tchan := make([]string, 0, len(robot.plugChannels))
	tchan = append(tchan, robot.plugChannels...)
	externalScripts := make([]externalScript, 0, len(robot.externalScripts))
	externalScripts = append(externalScripts, robot.externalScripts...)
	robot.RUnlock() // we're done with bot data 'til the end

	i := 0

	for plugname := range pluginHandlers {
		plugin := &botPlugin{
			pluginType: plugGo,
			botTask: botTask{
				name:   plugname,
				taskID: getTaskID(plugname),
			},
		}
		tlist = append(tlist, plugin)
		taskIndexByID[task.taskID] = i
		taskIndexByName[task.name] = i
		i++
	}

	for index, script := range externalScripts {
		if !taskNameRe.MatchString(script.Name) {
			Log(Error, fmt.Sprintf("Task name: '%s', index: %d doesn't match task name regex '%s', skipping", script.Name, index+1, taskNameRe.String()))
			continue
		}
		if script.Name == "bot" {
			Log(Error, "Illegal task name: bot - skipping")
			continue
		}
		if dup, ok := taskIndexByName[script.Name]; ok {
			msg := fmt.Sprintf("External script index: #%d, name: '%s' duplicates name of builtIn or Go plugin, skipping", index, script.Name)
			Log(Error, msg)
			r.debug(tlist[dup].taskID, msg, false)
			continue
		}
		t := &botTask{
			name:       script.Name,
			taskID:     getTaskID(script.Name),
			scriptPath: script.Path,
		}
		if len(task.Path) == 0 {
			msg := fmt.Sprintf("Task '%s' has zero-length path, disabling", task.Name)
			Log(Error, msg)
			r.debug(task.taskID, msg, false)
			t.Disabled = true
			t.reason = msg
		}
		switch task.Type {
		case "job", "Job":
			j := &botJob{
				botTask: task,
			}
			tlist = append(tlist, j)
		case "plugin", "Plugin":
			p := &botPlugin{
				pluginType: plugExternal,
				botTask:    task,
			}
			tlist = append(tlist, j)
		default:
			Log(Error, fmt.Sprintf("Task '%s' has unknown type '%s', should be one of job|plugin", task.Name, task.Type))
			continue
		}
		taskIndexByID[task.taskID] = i
		taskIndexByName[task.name] = i
		i++
	}

	// Load configuration for all valid plugins. Note that this is all being loaded
	// in to non-shared data structures that will replace current configuration
	// under lock at the end.
LoadLoop:
	for i, j := range tlist {
		var plugin *botPlugin
		var job *botJob
		var task *botTask
		var isPlugin bool
		switch t := j.(type) {
		case *botPlugin:
			isPlugin = true
			plug = t
			task = t.botTask
		case *botJob:
			job = t
			task = t.botTask
		}

		if task.Disabled {
			continue
		}
		tcfgload := make(map[string]json.RawMessage)
		Log(Debug, fmt.Sprintf("Loading configuration for task #%d - %s, type %d", i, task.name, plugin.pluginType))

		if isPlugin {
			if plugin.pluginType == plugExternal {
				// External plugins spit their default config to stdout when called with command="configure"
				cfg, err := getExtDefCfg(task)
				if err != nil {
					msg := fmt.Sprintf("Error getting default configuration for external plugin, disabling: %v", err)
					Log(Error, msg)
					r.debug(task.taskID, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
				if len(*cfg) > 0 {
					r.debug(task.taskID, fmt.Sprintf("Loaded default config from the plugin, size: %d", len(*cfg)), false)
				} else {
					r.debug(task.taskID, "Unable to obtain default config from plugin, command 'configure' returned no content", false)
				}
				if err := yaml.Unmarshal(*cfg, &tcfgload); err != nil {
					msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
					Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for '%s', disabling: %v", task.name, err))
					r.debug(task.taskID, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			} else {
				if err := yaml.Unmarshal([]byte(pluginHandlers[task.name].DefaultConfig), &pcfgload); err != nil {
					msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
					Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for '%s', disabling: %v", task.name, err))
					r.debug(task.taskID, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			}
		}
		// getConfigFile overlays the default config with configuration from the install path, then config path
		cpath := "jobs/"
		if isPlugin {
			cpath = "plugins/"
		}
		if err := r.getConfigFile(cpath+task.name+".yaml", task.taskID, false, tcfgload); err != nil {
			msg := fmt.Sprintf("Problem loading configuration file(s) for task '%s', disabling: %v", task.name, err)
			Log(Error, msg)
			r.debug(task.taskID, msg, false)
			task.Disabled = true
			task.reason = msg
			continue
		}
		if disjson, ok := pcfgload["Disabled"]; ok {
			disabled := false
			if err := json.Unmarshal(disjson, &disabled); err != nil {
				msg := fmt.Sprintf("Problem unmarshalling value for 'Disabled' in plugin '%s', disabling: %v", task.name, err)
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			}
			if disabled {
				msg := fmt.Sprintf("Plugin '%s' is disabled by configuration", task.name)
				Log(Info, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			}
		}
		// Boolean false values can be explicitly false, or default to false
		// when not specified. In some cases that matters.
		explicitAllChannels := false
		explicitAllowDirect := false
		explicitDenyDirect := false
		denyDirect := false

		for key, value := range tcfgload {
			var strval string
			var intval int
			var boolval bool
			var sarrval []string
			var hval []PluginHelp
			var mval []InputMatcher
			var val interface{}
			skip := false
			switch key {
			case "Description", "Elevator", "Authorizer", "AuthRequire", "NameSpace", "Channel", "Notify":
				val = &strval
			case "MaxHistories":
				val = &intval
			case "Disabled", "AllowDirect", "DirectOnly", "DenyDirect", "AllChannels", "RequireAdmin", "AuthorizeAllCommands", "CatchAll", "Verbose":
				val = &boolval
			case "Channels", "ElevatedCommands", "ElevateImmediateCommands", "Users", "AuthorizedCommands", "AdminCommands", "RequiredParameters":
				val = &sarrval
			case "Help":
				val = &hval
			case "CommandMatchers", "ReplyMatchers", "MessageMatchers", "Triggers":
				val = &mval
			case "Config":
				skip = true
			default:
				msg := fmt.Sprintf("Invalid configuration key for task '%s': %s - disabling", task.name, key)
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}

			if !skip {
				if err := json.Unmarshal(value, val); err != nil {
					msg := fmt.Sprintf("Disabling plugin '%s' - error unmarshalling value '%s': %v", task.name, key, err)
					Log(Error, msg)
					r.debug(task.taskID, msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
			}

			mismatch := false
			switch key {
			case "AllowDirect":
				task.AllowDirect = *(val.(*bool))
				explicitAllowDirect = true
			case "DirectOnly":
				task.DirectOnly = *(val.(*bool))
			case "Channels":
				task.Channels = *(val.(*[]string))
			case "AllChannels":
				task.AllChannels = *(val.(*bool))
				explicitAllChannels = true
			case "RequireAdmin":
				task.RequireAdmin = *(val.(*bool))
			case "AdminCommands":
				if isPlugin {
					plugin.AdminCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "Description":
				task.Description = *(val.(*string))
			case "Elevator":
				task.Elevator = *(val.(*string))
			case "ElevatedCommands":
				if isPlugin {
					plugin.ElevatedCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "ElevateImmediateCommands":
				if isPlugin {
					plugin.ElevateImmediateCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "Users":
				task.Users = *(val.(*[]string))
			case "Authorizer":
				task.Authorizer = *(val.(*string))
			case "AuthRequire":
				task.AuthRequire = *(val.(*string))
			case "AuthorizedCommands":
				if isPlugin {
					plugin.AuthorizedCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "AuthorizeAllCommands":
				if isPlugin {
					plugin.AuthorizeAllCommands = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "Help":
				if isPlugin {
					plugin.Help = *(val.(*[]PluginHelp))
				} else {
					mismatch = true
				}
			case "CommandMatchers":
				if isPlugin {
					plugin.CommandMatchers = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "ReplyMatchers":
				if isPlugin {
					task.ReplyMatchers = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "MessageMatchers":
				if isPlugin {
					plugin.MessageMatchers = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "CatchAll":
				if isPlugin {
					plugin.CatchAll = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "Channel":
				if isPlugin {
					mismatch = true
				} else {
					job.Channel = *(val.(*string))
				}
			case "Notify":
				if isPlugin {
					mismatch = true
				} else {
					job.Notify = *(val.(*string))
				}
			case "Verbose":
				if isPlugin {
					mismatch = true
				} else {
					job.Verbose = *(val.(*bool))
				}
			case "Triggers":
				if isPlugin {
					mismatch = true
				} else {
					job.Triggers = *(val.(*[]InputMatcher))
				}
			case "RequiredParameters":
				if isPlugin {
					mismatch = true
				} else {
					job.RequiredParameters = *(val.(*[]string))
				}
			case "Config":
				task.Config = value
			}
			if mismatch {
				if isPlugin {
					msg := fmt.Sprintf("Disabling plugin '%s' - invalid configuration key: %s", task.name, key)
				} else {
					msg := fmt.Sprintf("Disabling job '%s' - invalid configuration key: %s", task.name, key)
				}
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
		}
		// End of reading configuration keys

		// Start sanity checking of configuration
		if task.DirectOnly {
			if explicitAllowDirect {
				if !task.AllowDirect {
					msg := fmt.Sprintf("Task '%s' has conflicting values for AllowDirect (false) and DirectOnly (true), disabling", task.name)
					Log(Error, msg)
					r.debug(task.taskID, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			} else {
				Log(Debug, "DirectOnly specified without AllowDirect; setting AllowDirect = true")
				task.AllowDirect = true
				explicitAllowDirect = true
			}
		}

		if !explicitAllowDirect {
			task.AllowDirect = defaultAllowDirect
		}

		// Use bot default plugin channels if none defined, unless AllChannels requested.
		if len(task.Channels) == 0 {
			if len(tchan) > 0 {
				if !task.AllChannels { // AllChannels = true is always explicit
					task.Channels = tchan
				}
			} else { // no default channels specified
				if !explicitAllChannels { // if AllChannels wasn't explicitly configured, and no default channels, default to AllChannels = true
					task.AllChannels = true
				}
			}
		}
		// Note: you can't combine the channel length checking logic, the above
		// can change it.

		// Considering possible default channels, is the plugin visible anywhere?
		if len(task.Channels) > 0 {
			msg := fmt.Sprintf("Task '%s' will be available in channels %q", task.name, task.Channels)
			Log(Info, msg)
			r.debug(task.taskID, msg, false)
		} else {
			if !(task.AllowDirect || task.AllChannels) {
				msg := fmt.Sprintf("Task '%s' not visible in any channels or by direct message, disabling", task.name)
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			} else {
				msg := fmt.Sprintf("Task '%s' has no channel restrictions configured; all channels: %t", task.name, task.AllChannels)
				Log(Info, msg)
				r.debug(task.taskID, msg, false)
			}
		}

		// Compile the regex's
		for i := range plugin.CommandMatchers {
			command := &plugin.CommandMatchers[i]
			regex := massageRegexp(command.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				msg := fmt.Sprintf("Disabling %s, couldn't compile command regular expression '%s': %v", task.name, regex, err)
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			} else {
				command.re = re
			}
		}
		for i := range task.Triggers {
			trigger := &task.Triggers[i]
			regex := massageRegexp(trigger.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				msg := fmt.Sprintf("Disabling %s, couldn't compile trigger regular expression '%s': %v", task.name, regex, err)
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			} else {
				trigger.re = re
			}
		}
		for i := range task.ReplyMatchers {
			reply := &task.ReplyMatchers[i]
			regex := massageRegexp(reply.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				msg := fmt.Sprintf("Skipping %s, couldn't compile reply regular expression '%s': %v", task.name, regex, err)
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			} else {
				reply.re = re
			}
		}
		for i := range plugin.MessageMatchers {
			// Note that full message regexes don't get the beginning and end anchors added - the individual plugin
			// will need to do this if necessary.
			message := &plugin.MessageMatchers[i]
			regex := massageRegexp(message.Regex)
			re, err := regexp.Compile(regex)
			if err != nil {
				msg := fmt.Sprintf("Skipping %s, couldn't compile message regular expression '%s': %v", task.name, regex, err)
				Log(Error, msg)
				r.debug(task.taskID, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			} else {
				message.re = re
			}
		}

		// Make sure all security-related command lists resolve to actual
		// commands to guard against typos.
		cmdlist := []struct {
			ctype string
			clist []string
		}{
			{"elevated", plugin.ElevatedCommands},
			{"elevate immediate", plugin.ElevateImmediateCommands},
			{"authorized", plugin.AuthorizedCommands},
			{"admin", plugin.AdminCommands},
		}
		for _, cmd := range cmdlist {
			if len(cmd.clist) > 0 {
				for _, i := range cmd.clist {
					cmdfound := false
					for _, j := range plugin.CommandMatchers {
						if i == j.Command {
							cmdfound = true
							break
						}
					}
					if !cmdfound {
						for _, j := range plugin.MessageMatchers {
							if i == j.Command {
								cmdfound = true
								break
							}
						}
					}
					if !cmdfound {
						msg := fmt.Sprintf("Disabling %s, %s command %s didn't match a command from CommandMatchers or MessageMatchers", task.name, cmd.ctype, i)
						Log(Error, msg)
						r.debug(task.taskID, msg, false)
						task.Disabled = true
						task.reason = msg
						continue LoadLoop
					}
				}
			}
		}

		// For Go plugins, use the provided empty config struct to go ahead
		// and unmarshall Config. The GetTaskConfig call just sets a pointer
		// without unmshalling again.
		if plugin.pluginType == plugGo {
			// Copy the pointer to the empty config struct / empty struct (when no config)
			// pluginHandlers[name].Config is an empty struct for unmarshalling provided
			// in RegisterPlugin.
			pt := reflect.ValueOf(pluginHandlers[task.name].Config)
			if pt.Kind() == reflect.Ptr {
				if task.Config != nil {
					// reflect magic: create a pointer to a new empty config struct for the plugin
					task.config = reflect.New(reflect.Indirect(pt).Type()).Interface()
					if err := json.Unmarshal(task.Config, task.config); err != nil {
						msg := fmt.Sprintf("Error unmarshalling plugin config json to config, disabling: %v", err)
						Log(Error, msg)
						r.debug(task.taskID, msg, false)
						task.Disabled = true
						task.reason = msg
						continue
					}
				} else {
					// Providing custom config not required (should it be?)
					msg := fmt.Sprintf("Plugin '%s' has custom config, but none is configured", task.name)
					Log(Warn, msg)
					r.debug(task.taskID, msg, false)
				}
			} else {
				if task.Config != nil {
					msg := fmt.Sprintf("Custom configuration data provided for Go plugin '%s', but no config struct was registered; disabling", task.name)
					Log(Error, msg)
					r.debug(task.taskID, msg, false)
					task.Disabled = true
					task.reason = msg
				} else {
					Log(Debug, fmt.Sprintf("Config interface isn't a pointer, skipping unmarshal for Go plugin '%s'", task.name))
				}
			}
		}
		Log(Debug, fmt.Sprintf("Configured plugin #%d, '%s'", i, task.name))
	}
	// End of configuration loading. All invalid tasks are disabled.

	reInitPlugins := false
	currentTasks.Lock()
	currentTasks.t = &tlist
	currentTasks.idMap = &taskIndexByID
	currentTasks.nameMap = &taskIndexByName
	currentTasks.Unlock()
	// loadTaskConfig is called in initBot, before the connector has started;
	// don't init plugins in that case.
	robot.RLock()
	if robot.Connector != nil {
		reInitPlugins = true
	}
	robot.RUnlock()
	if reInitPlugins {
		initializePlugins()
	}
}
