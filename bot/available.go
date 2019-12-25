package bot

import (
	"fmt"
	"path/filepath"
	"strings"
)

// pluginAvailable checks the user and channel against the task's
// configuration to determine if the task should be available. Used by
// both handleMessage and the help builtin. verboseOnly is set when availability
// is being checked for ambient messages or auth/elevation plugins, to indicate
// debugging verboseness.
func (r Robot) pluginAvailable(task *Task, helpSystem, verboseOnly bool) (available bool) {
	nvmsg := "task is NOT visible to user " + r.User + " in channel "
	vmsg := "task is visible to user " + r.User + " in channel "
	if r.directMsg {
		nvmsg += "(direct message)"
		vmsg += "(direct message)"
	} else {
		nvmsg += r.Channel
		vmsg += r.Channel
	}
	defer func(vmsg string) {
		if available {
			debugTask(task, vmsg, verboseOnly)
		}
	}(vmsg)
	if task.Disabled {
		debugTask(task, nvmsg+"; task is disabled, possibly due to configuration error", verboseOnly)
		return false
	}
	if !r.directMsg && task.DirectOnly && !helpSystem {
		debugTask(task, nvmsg+"; only available by direct message: DirectOnly is TRUE", verboseOnly)
		return false
	}
	if r.directMsg && !task.AllowDirect && !helpSystem {
		debugTask(task, nvmsg+"; not available by direct message: AllowDirect is FALSE", verboseOnly)
		return false
	}
	if task.RequireAdmin {
		isAdmin := false
		admins := r.cfg.adminUsers
		for _, adminUser := range admins {
			if r.User == adminUser {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			debugTask(task, nvmsg+"; RequireAdmin is TRUE and user isn't an Admin", verboseOnly)
			return false
		}
	}
	if len(task.Users) > 0 {
		userOk := false
		for _, allowedUser := range task.Users {
			match, err := filepath.Match(allowedUser, r.User)
			if match && err == nil {
				userOk = true
			}
		}
		if !userOk {
			debugTask(task, nvmsg+"; user is not on the list of allowed users", verboseOnly)
			return false
		}
	}
	if r.directMsg && (task.AllowDirect || task.DirectOnly) {
		return true
	}
	if len(task.Channels) > 0 {
		for _, pchannel := range task.Channels {
			if pchannel == r.Channel {
				return true
			}
		}
	} else {
		if task.AllChannels {
			return true
		}
	}
	if helpSystem {
		return true
	}
	debugTask(task, fmt.Sprintf(nvmsg+"; channel '%s' is not on the list of allowed channels: %s", r.Channel, strings.Join(task.Channels, ", ")), verboseOnly)
	return false
}
