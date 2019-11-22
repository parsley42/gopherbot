// +build !modular

package slack

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPlugin("slackutil", slackplugin)
}

func init() {
	bot.RegisterConnector("slack", Initialize)
}
