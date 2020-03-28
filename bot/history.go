package bot

/*
	history.go provides the mechanism and methods for storing and retrieving
	job / plugin run histories of stdout/stderr for a given run. Each time
	a job / plugin is initiated by a trigger, scheduled job, or user command,
	a new history file is started if HistoryLogs is != 0 for the job/plugin.
	The history provider will store histories up to some maximum, and return
	that history based on the index.
*/

import (
	"log"
	"time"

	"github.com/lnxjedi/robot"
)

const histPrefix = "bot:histories:"

// Memory that holds a Ref -> historyLookup record
const histLookup = "bot:histories-lookup"

type historyLog struct {
	LogIndex   int
	Ref        string // 6 hex digits from worker ID
	CreateTime string
}

type historyLookup struct {
	Tag   string
	Index int
}

type pipeHistory struct {
	NextIndex          int
	Histories          []historyLog
	ExtendedNamespaces []string
}

// start a new history log and manage memories
/*
Args:
- tag: pipeline name or job:extended_namespace
- eid: 8 random hex digits generated in registerActive, for lookups

*/
func newHistory(tag, eid string, wid, keep int) (logger robot.HistoryLogger, url string, idx int, err error) {
	var start time.Time
	currentCfg.RLock()
	tz := currentCfg.timeZone
	currentCfg.RUnlock()
	if tz != nil {
		start = time.Now().In(tz)
	} else {
		start = time.Now()
	}
	// checkout memories and figure out idx
	// TODO: generate idx before using it !!!
	hist := historyLog{
		LogIndex:   idx,
		Ref:        eid,
		CreateTime: start.Format("Mon Jan 2 15:04:05 MST 2006"),
	}

	if keep > 0 {
		url, _ = interfaces.history.GetLogURL(tag, idx)
	}
	return
}

// Map of registered history providers
var historyProviders = make(map[string]func(robot.Handler) robot.HistoryProvider)

// RegisterHistoryProvider allows history implementations to register a function
// with a named provider type that returns a HistoryProvider interface.
func RegisterHistoryProvider(name string, provider func(robot.Handler) robot.HistoryProvider) {
	if stopRegistrations {
		return
	}
	if historyProviders[name] != nil {
		log.Fatal("Attempted registration of duplicate history provider name:", name)
	}
	historyProviders[name] = provider
}
