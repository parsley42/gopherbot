package bot

import b "github.com/lnxjedi/gopherbot/models"

const technicalElevError = "Sorry, elevation failed due to a problem with the elevation service"
const configElevError = "Sorry, elevation failed due to a configuration error"

// Elevator plugins provide an elevate method for checking if the user
// can run a privileged command.

func (c *botContext) elevate(task *BotTask, immediate bool) (retval b.TaskRetVal) {
	r := c.makeRobot()
	botCfg.RLock()
	defaultElevator := botCfg.defaultElevator
	botCfg.RUnlock()
	if task.Elevator == "" && defaultElevator == "" {
		Log(Audit, "Task '%s' requires elevation, but no elevator configured", task.name)
		r.Say(configElevError)
		emit(ElevNoRunMisconfigured)
		return b.ConfigurationError
	}
	elevator := defaultElevator
	if task.Elevator != "" {
		elevator = task.Elevator
	}
	_, ePlug, _ := getTask(c.tasks.getTaskByName(elevator))
	if ePlug != nil {
		immedString := "true"
		if !immediate {
			immedString = "false"
		}
		_, elevRet := c.callTask(ePlug, "elevate", immedString)
		if elevRet == b.Success {
			Log(Audit, "Elevation succeeded by elevator '%s', user '%s', task '%s' in channel '%s'", ePlug.name, c.User, task.name, c.Channel)
			emit(ElevRanSuccess)
			return b.Success
		}
		if elevRet == b.Fail {
			Log(Audit, "Elevation FAILED by elevator '%s', user '%s', task '%s' in channel '%s'", ePlug.name, c.User, task.name, c.Channel)
			r.Say("Sorry, this command requires elevation")
			emit(ElevRanFail)
			return b.Fail
		}
		if elevRet == b.MechanismFail {
			Log(Audit, "Elevator plugin '%s' mechanism failure while elevating user '%s' for task '%s' in channel '%s'", ePlug.name, c.User, task.name, c.Channel)
			r.Say(technicalElevError)
			emit(ElevRanMechanismFailed)
			return b.MechanismFail
		}
		if elevRet == b.Normal {
			Log(Audit, "Elevator plugin '%s' returned 'Normal' (0) instead of 'Success' (1), failing elevation in '%s' for task '%s' in channel '%s'", ePlug.name, c.User, task.name, c.Channel)
			r.Say(technicalElevError)
			emit(ElevRanFailNormal)
			return b.MechanismFail
		}
		Log(Audit, "Elevator plugin '%s' exit code %d while elevating user '%s' for task '%s' in channel '%s'", ePlug.name, retval, c.User, task.name, c.Channel)
		r.Say(technicalElevError)
		emit(ElevRanFailOther)
		return b.MechanismFail
	}
	Log(Audit, "Elevator plugin '%s' not found while elevating user '%s' for task '%s' in channel '%s'", task.Elevator, c.User, task.name, c.Channel)
	r.Say(technicalElevError)
	emit(ElevNoRunNotFound)
	return b.ConfigurationError
}

// Check for a configured Elevator and check elevation
func (c *botContext) checkElevation(t interface{}, command string) (retval b.TaskRetVal, required bool) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	immediate := false
	elevationRequired := false
	if isPlugin && len(plugin.ElevateImmediateCommands) > 0 {
		for _, i := range plugin.ElevateImmediateCommands {
			if command == i {
				elevationRequired = true
				immediate = true
				break
			}
		}
	}
	if isPlugin && !elevationRequired && len(plugin.ElevatedCommands) > 0 {
		for _, i := range plugin.ElevatedCommands {
			if command == i {
				elevationRequired = true
				break
			}
		}
	}
	if !isPlugin {
		if len(task.Elevator) > 0 {
			elevationRequired = true
		}
	}
	if !elevationRequired {
		return b.Success, false
	}
	retval = c.elevate(task, immediate)
	if retval == b.Success {
		return b.Success, true
	}
	Log(Error, "Elevation failed for task '%s', command: '%s'", task.name, command)
	return b.Fail, true
}
