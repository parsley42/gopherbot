package bot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type JSONFunction struct {
	FuncName string
	User     string
	Channel  string
	Format   string
	PluginID string
	FuncArgs json.RawMessage
}

type attribute struct {
	Attribute string
}

type userattr struct {
	User      string
	Attribute string
}

type logmessage struct {
	Level   string
	Message string
}

type channelmessage struct {
	Channel string
	Message string
	Format  string
}

type usermessage struct {
	User    string
	Message string
	Format  string
}

type userchannelmessage struct {
	User    string
	Channel string
	Message string
	Format  string
}

type replyrequest struct {
	RegExId string
	Timeout int
}

// Types for returning values
type attrresponse struct {
	Attribute string
	BotRetVal int
}

type botretvalresponse struct {
	BotRetVal int
}

type waitreplyresponse struct {
	Reply     string
	BotRetVal int
}

func (b *robot) listenHttpJSON() {
	if len(b.port) > 0 {
		http.Handle("/json", b)
		b.Log(Fatal, http.ListenAndServe(b.port, nil))
	}
}

// decode looks for a base64: prefix, then removes it and tries to decode the message
func (b *robot) decode(msg string) string {
	if strings.HasPrefix(msg, "base64:") {
		msg = strings.TrimPrefix(msg, "base64:")
		decoded, err := base64.StdEncoding.DecodeString(msg)
		if err != nil {
			b.Log(Error, fmt.Errorf("Unable to decode base64 message %s: %v", msg, err))
			return msg
		}
		return string(decoded)
	} else {
		return msg
	}
}

func encode(arg string) string {
	return "base64:" + base64.StdEncoding.EncodeToString([]byte(arg))
}

func (b *robot) getArgs(rw http.ResponseWriter, jsonargs *json.RawMessage, args interface{}) bool {
	err := json.Unmarshal(*jsonargs, args)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		b.Log(Error, "Couldn't deciper JSON args: ", err)
		return false
	}
	return true
}

func (b *robot) sendReturn(rw http.ResponseWriter, ret interface{}) {
	d, err := json.Marshal(ret)
	if err != nil { // this should never happen
		b.Log(Fatal, fmt.Sprintf("BUG in bot/http.go:sendReturn, error marshalling JSON: %v", err))
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(d)
}

func (b *robot) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.Log(Fatal, err)
	}
	defer r.Body.Close()

	var f JSONFunction
	err = json.Unmarshal(data, &f)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		b.Log(Error, "Couldn't decipher JSON command: ", err)
		return
	}

	// Generate a synthetic Robot for access to it's methods
	bot := Robot{
		User:     f.User,
		Channel:  f.Channel,
		Format:   setFormat(f.Format),
		pluginID: f.PluginID,
		robot:    b,
	}

	var (
		attr, reply string
		ret         BotRetVal
	)
	switch f.FuncName {
	case "GetSenderAttribute", "GetBotAttribute":
		var a attribute
		if !b.getArgs(rw, &f.FuncArgs, &a) {
			return
		}
		if f.FuncName == "GetBotAttribute" {
			attr, ret = bot.GetBotAttribute(a.Attribute)
		} else {
			attr, ret = bot.GetSenderAttribute(a.Attribute)
		}
		b.sendReturn(rw, &attrresponse{encode(attr), int(ret)})
		return
	case "GetUserAttribute":
		var ua userattr
		if !b.getArgs(rw, &f.FuncArgs, &ua) {
			return
		}
		attr, ret = bot.GetUserAttribute(ua.User, ua.Attribute)
		b.sendReturn(rw, &attrresponse{encode(attr), int(ret)})
		return
	case "LogMessage":
		var lm logmessage
		if !b.getArgs(rw, &f.FuncArgs, &lm) {
			return
		}
		l := b.logStrToLevel(lm.Level)
		b.Log(l, lm.Message)
		b.sendReturn(rw, &botretvalresponse{int(Ok)})
		return
	case "SendChannelMessage":
		var cm channelmessage
		if !b.getArgs(rw, &f.FuncArgs, &cm) {
			return
		}
		b.sendReturn(rw, &botretvalresponse{
			int(bot.SendChannelMessage(cm.Channel, b.decode(cm.Message))),
		})
		return
	case "SendUserChannelMessage":
		var ucm userchannelmessage
		if !b.getArgs(rw, &f.FuncArgs, &ucm) {
			return
		}
		b.sendReturn(rw, &botretvalresponse{
			int(bot.SendUserChannelMessage(ucm.User, ucm.Channel, b.decode(ucm.Message))),
		})
		return
	case "SendUserMessage":
		var um usermessage
		if !b.getArgs(rw, &f.FuncArgs, &um) {
			return
		}
		b.sendReturn(rw, &botretvalresponse{
			int(bot.SendUserMessage(um.User, b.decode(um.Message))),
		})
		return
	case "WaitForReply":
		var rr replyrequest
		if !b.getArgs(rw, &f.FuncArgs, &rr) {
			return
		}
		reply, ret = bot.WaitForReply(rr.RegExId, rr.Timeout)
		b.sendReturn(rw, &waitreplyresponse{encode(reply), int(ret)})
		return
	// NOTE: "Say" and "Reply" are implemented in shellLib.sh or other scripting library
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
}
