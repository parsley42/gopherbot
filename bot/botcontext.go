package bot

import (
	"crypto/rand"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

/* botcontext.go - internal methods on botContexts */

/* NOTE on variable conventions:
c is used for passing a context between methods, mostly to be read
ctx is used for a context being modified or requiring complex locking
cu is used for an unlocked context where only unchanging values are read
*/

// Global robot run number (incrementing int)
var botRunID = struct {
	idx int
	sync.Mutex
}{
	0,
	sync.Mutex{},
}

// Global persistent maps of pipelines running, for listing/forcibly
// stopping.
var activePipelines = struct {
	i    map[int]*botContext
	eids map[string]struct{}
	sync.Mutex
}{
	make(map[int]*botContext),
	make(map[string]struct{}),
	sync.Mutex{},
}

// getBotContextInt is used to look up a botContext from a Robot in when needed.
// Note that 0 is never a valid bot id, and this will return nil in that case.
func getBotContextInt(idx int) *botContext {
	activePipelines.Lock()
	bot, _ := activePipelines.i[idx]
	activePipelines.Unlock()
	return bot
}

// Assign a bot run number and register it in the global map of running
// pipelines.
func (c *botContext) registerActive(parent *botContext) {
	if c.Incoming != nil {
		c.Protocol, _ = getProtocol(c.Incoming.Protocol)
	}
	c.Format = currentCfg.defaultMessageFormat
	c.environment["GOPHER_HTTP_POST"] = "http://" + listenPort

	// Only needed for bots not created by IncomingMessage
	if c.maps == nil {
		currentUCMaps.Lock()
		c.maps = currentUCMaps.ucmap
		currentUCMaps.Unlock()
	}
	if len(c.ProtocolUser) == 0 && len(c.User) > 0 {
		if idRegex.MatchString(c.User) {
			c.ProtocolUser = c.User
		} else if ui, ok := c.maps.user[c.User]; ok {
			c.ProtocolUser = bracket(ui.UserID)
			c.BotUser = ui.BotUser
		} else {
			c.ProtocolUser = c.User
		}
	}
	if len(c.ProtocolChannel) == 0 && len(c.Channel) > 0 {
		if idRegex.MatchString(c.Channel) {
			c.ProtocolChannel = c.Channel
		} else if ci, ok := c.maps.channel[c.Channel]; ok {
			c.ProtocolChannel = bracket(ci.ChannelID)
		} else {
			c.ProtocolChannel = c.Channel
		}
	}

	c.nextTasks = make([]TaskSpec, 0)
	c.finalTasks = make([]TaskSpec, 0)

	c.environment["GOPHER_INSTALLDIR"] = installPath

	botRunID.Lock()
	botRunID.idx++
	if botRunID.idx == 0 {
		botRunID.idx = 1
	}
	c.id = botRunID.idx
	botRunID.Unlock()
	var eid string
	activePipelines.Lock()
	for {
		// 4 bytes of entropy per pipeline
		b := make([]byte, 4)
		rand.Read(b)
		eid = fmt.Sprintf("%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
		if _, ok := activePipelines.eids[eid]; !ok {
			activePipelines.eids[eid] = struct{}{}
			break
		}
	}
	c.environment["GOPHER_CALLER_ID"] = eid
	c.eid = eid

	if parent != nil {
		parent._child = c
		c._parent = parent
	}
	activePipelines.i[c.id] = c
	activePipelines.Unlock()
	c.active = true
}

// deregister must be called for all registered Robots to prevent a memory leak.
func (c *botContext) deregister() {
	activePipelines.Lock()
	delete(activePipelines.i, c.id)
	delete(activePipelines.eids, c.eid)
	activePipelines.Unlock()
	c.active = false
}

// makeRobot returns a *Robot for plugins; the id lets Robot methods
// get a reference back to the original context when needed. The Robot
// should contain a copy of almost all of the information needed for plugins
// to run.
func (c *botContext) makeRobot() *Robot {
	c.Lock()
	r := &Robot{
		Message: &robot.Message{
			User:            c.User,
			ProtocolUser:    c.ProtocolUser,
			Channel:         c.Channel,
			ProtocolChannel: c.ProtocolChannel,
			Format:          c.Format,
			Protocol:        c.Protocol,
			Incoming:        c.Incoming,
		},
		eid:           c.eid,
		id:            c.id,
		automaticTask: c.automaticTask,
		directMsg:     c.directMsg,
		currentTask:   c.currentTask,
		nsExtension:   c.nsExtension,
		cfg:           c.cfg,
		tasks:         c.tasks,
	}
	c.Unlock()
	return r
}

// clone() is a convenience function to clone the current context before
// starting a new goroutine for startPipeline. Used by e.g. triggered jobs,
// SpawnJob(), and runPipeline for sub-jobs.
func (c *botContext) clone() *botContext {
	c.Lock()
	clone := &botContext{
		User:             c.User,
		ProtocolUser:     c.ProtocolUser,
		Channel:          c.Channel,
		ProtocolChannel:  c.ProtocolChannel,
		Incoming:         c.Incoming,
		directMsg:        c.directMsg,
		BotUser:          c.BotUser,
		listedUser:       c.listedUser,
		pipeName:         c.pipeName,
		pipeDesc:         c.pipeDesc,
		ptype:            c.ptype,
		cfg:              c.cfg,
		tasks:            c.tasks,
		maps:             c.maps,
		repositories:     c.repositories,
		automaticTask:    c.automaticTask,
		elevated:         c.elevated,
		Protocol:         c.Protocol,
		Format:           c.Format,
		msg:              c.msg,
		workingDirectory: "",
		environment:      make(map[string]string),
	}
	c.Unlock()
	return clone
}

// botContext is created for each incoming message, in a separate goroutine that
// persists for the life of the message, until finally a plugin runs
// (or doesn't). It could also be called Context, or PipelineState; but for
// use by plugins, it's best left as Robot.
type botContext struct {
	sync.Mutex                                   // Lock to protect the bot context when pipeline running
	User             string                      // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel          string                      // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	ProtocolUser     string                      // The username or <userid> to be sent in connector methods
	ProtocolChannel  string                      // the channel name or <channelid> where the message originated
	Protocol         robot.Protocol              // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	Incoming         *robot.ConnectorMessage     // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	Format           robot.MessageFormat         // robot's default message format
	workingDirectory string                      // directory where tasks run relative to $(pwd)
	baseDirectory    string                      // base for this pipeline relative to $(pwd), depends on `Homed`, affects SetWorkingDirectory
	privileged       bool                        // privileged jobs flip this flag, causing tasks in the pipeline to run in cfgdir
	id               int                         // incrementing index of Robot threads
	eid              string                      // unique ID for external tasks
	tasks            *taskList                   // Pointers to current task configuration at start of pipeline
	maps             *userChanMaps               // Pointer to current user / channel maps struct
	repositories     map[string]robot.Repository // Set of configured repositories
	cfg              *configuration              // Active configuration when this context was created
	BotUser          bool                        // set for bots/programs that should never match ambient messages
	listedUser       bool                        // set for users listed in the UserRoster; ambient messages don't match unlisted users by default
	isCommand        bool                        // Was the message directed at the robot, dm or by mention
	directMsg        bool                        // if the message was sent by DM
	msg              string                      // the message text sent
	automaticTask    bool                        // set for scheduled & triggers jobs, where user security restrictions don't apply
	history          robot.HistoryProvider       // history provider for generating the logger
	timeZone         *time.Location              // for history timestamping
	logger           robot.HistoryLogger         // where to send stdout / stderr
	active           bool                        // whether this context has been registered as active
	ptype            pipelineType                // what started this pipeline

	// Parent and child values protected by the activePipelines lock
	_parent, _child *botContext
	elevated        bool              // set when required elevation succeeds
	environment     map[string]string // environment vars set for each job/plugin in the pipeline
	//taskenvironment map[string]string // per-task environment for Go plugins
	stage          pipeStage  // which pipeline is being run; primaryP, finalP, failP
	jobInitialized bool       // whether a job has started
	jobName        string     // name of the running job
	jobChannel     string     // channel where job updates are posted
	nsExtension    string     // extended namespace
	runIndex       int        // run number of a job
	verbose        bool       // flag if initializing job was verbose
	nextTasks      []TaskSpec // tasks in the pipeline
	finalTasks     []TaskSpec // clean-up tasks that always run when the pipeline ends
	failTasks      []TaskSpec // clean-up tasks that run when a pipeline fails

	failedTask, failedTaskDescription string // set when a task fails

	pipeName, pipeDesc string      // name and description of task that started pipeline
	currentTask        interface{} // pointer to currently executing task
	taskName           string      // name of current task
	taskDesc           string      // description for same
	osCmd              *exec.Cmd   // running Command, for aborting a pipeline

	exclusiveTag  string // tasks with the same exclusiveTag never run at the same time
	exclusive     bool   // indicates task was running exclusively
	queueTask     bool   // whether to queue up if Exclusive call failed
	abortPipeline bool   // Exclusive request failed w/o queueTask
}
