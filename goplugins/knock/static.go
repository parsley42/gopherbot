// +build !modular

package knock

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPlugin("knock", knockhandler)
}
