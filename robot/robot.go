package robot

// Robot defines the methods exposed by gopherbot.bot Robot struct, for
// use by plugins.
type Robot interface {
	GetBotAttribute(a string) AttrRet
	GetUserAttribute(u, a string) AttrRet
	GetSenderAttribute(a string) AttrRet
	GetTaskConfig(dptr interface{}) RetVal
	SendChannelMessage(ch, msg string, v ...interface{}) RetVal
	SendUserChannelMessage(u, ch, msg string, v ...interface{}) RetVal
	SendUserMessage(u, msg string, v ...interface{}) RetVal
	Reply(msg string, v ...interface{}) RetVal
	Say(msg string, v ...interface{}) RetVal
	RandomInt(n int) int
	RandomString(s []string) string
	Pause(s float64)
	PromptForReply(regexID string, prompt string) (string, RetVal)
	PromptUserForReply(regexID string, user string, prompt string) (string, RetVal)
	PromptUserChannelForReply(regexID string, user string, channel string, prompt string) (string, RetVal)
}
