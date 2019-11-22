package slack

import (
	"regexp"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
	"github.com/nlopes/slack"
)

var idre = regexp.MustCompile(`slack id <@(.*)>`)

// Define the handler function
func slackutil(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	switch command {
	// This isn't really necessary
	case "init":
		// ignore
	case "identify":
		if m.Protocol != robot.Slack {
			r.Say("Sorry, that only works with Slack")
			return
		}
		sl := m.Incoming.MessageObject.(*slack.MessageEvent)
		sid := idre.FindStringSubmatch(sl.Text)[1]
		r.Say("User %s has Slack internal ID %s", args[0], sid)
	}
	return
}

func init() {
	bot.RegisterPlugin("slackutil", robot.PluginHandler{
		Handler: slackutil,
	})
}
