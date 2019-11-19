package bot

import b "github.com/lnxjedi/gopherbot/models"

const technicalAuthError = "Sorry, authorization failed due to a problem with the authorization plugin"
const configAuthError = "Sorry, authorization failed due to a configuration error"

// Check for a configured Authorizer and check authorization
func (c *botContext) checkAuthorization(t interface{}, command string, args ...string) (retval b.TaskRetVal) {
	task, plugin, _ := getTask(t)
	r := c.makeRobot()
	isPlugin := plugin != nil
	if isPlugin {
		if !(plugin.AuthorizeAllCommands || len(plugin.AuthorizedCommands) > 0) {
			// This plugin requires no authorization
			if task.Authorizer != "" {
				Log(Audit, "Plugin '%s' configured an authorizer, but has no commands requiring authorization", task.name)
				r.Say(configAuthError)
				return b.ConfigurationError
			}
			return b.Success
		} else if !plugin.AuthorizeAllCommands {
			authRequired := false
			for _, i := range plugin.AuthorizedCommands {
				if command == i {
					authRequired = true
					break
				}
			}
			if !authRequired {
				return b.Success
			}
		}
	} else {
		// Jobs don't have commands; only check authorization if an Authorizer
		// is explicitly set.
		if len(task.Authorizer) == 0 {
			return b.Success
		}
	}
	botCfg.RLock()
	defaultAuthorizer := botCfg.defaultAuthorizer
	botCfg.RUnlock()
	if isPlugin && task.Authorizer == "" && defaultAuthorizer == "" {
		Log(Audit, "Plugin '%s' requires authorization for command '%s', but no authorizer configured", task.name, command)
		r.Say(configAuthError)
		emit(AuthNoRunMisconfigured)
		return b.ConfigurationError
	}
	authorizer := defaultAuthorizer
	if task.Authorizer != "" {
		authorizer = task.Authorizer
	}
	authTask := c.tasks.getTaskByName(authorizer)
	if authTask == nil {
		return b.ConfigurationError
	}
	_, authPlug, _ := getTask(authTask)
	if authPlug != nil {
		args = append([]string{task.name, task.AuthRequire, command}, args...)
		_, authRet := c.callTask(authPlug, "authorize", args...)
		if authRet == b.Success {
			Log(Audit, "Authorization succeeded by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, c.User, command, task.name, c.Channel, task.AuthRequire)
			emit(AuthRanSuccess)
			return b.Success
		}
		if authRet == b.Fail {
			Log(Audit, "Authorization FAILED by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, c.User, command, task.name, c.Channel, task.AuthRequire)
			r.Say("Sorry, you're not authorized for that command")
			emit(AuthRanFail)
			return b.Fail
		}
		if authRet == b.MechanismFail {
			Log(Audit, "Auth plugin '%s' mechanism failure while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, c.User, command, task.name, c.Channel, task.AuthRequire)
			r.Say(technicalAuthError)
			emit(AuthRanMechanismFailed)
			return b.MechanismFail
		}
		if authRet == b.Normal {
			Log(Audit, "Auth plugin '%s' returned 'Normal' (%d) instead of 'Success' (%d), failing auth in '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, b.Normal, b.Success, c.User, command, task.name, c.Channel, task.AuthRequire)
			r.Say(technicalAuthError)
			emit(AuthRanFailNormal)
			return b.MechanismFail
		}
		Log(Audit, "Auth plugin '%s' exit code %s, failing auth while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, authRet, c.User, command, task.name, c.Channel, task.AuthRequire)
		r.Say(technicalAuthError)
		emit(AuthRanFailOther)
		return b.MechanismFail
	}
	Log(Audit, "Auth plugin '%s' not found while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", task.Authorizer, c.User, command, task.name, c.Channel, task.AuthRequire)
	r.Say(technicalAuthError)
	emit(AuthNoRunNotFound)
	return b.ConfigurationError
}
