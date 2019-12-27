package bot

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// GetMessage returns a pointer to the message struct
func (r Robot) GetMessage() *robot.Message {
	return r.Message
}

// GetUserAttribute returns a AttrRet with
// - The string Attribute of a user, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: supplement data with gopherbot.yaml user's table, if an
// admin wants to supplment whats available from the protocol.
func (r Robot) GetUserAttribute(u, a string) *robot.AttrRet {
	a = strings.ToLower(a)
	c := r.getLockedContext()
	defer c.Unlock()
	var user string
	var ui *UserInfo
	var ok bool
	if ui, ok = c.maps.user[u]; ok {
		user = "<" + ui.UserID + ">"
	} else {
		user = u
	}
	if ui != nil {
		var attr string
		switch a {
		case "name", "username", "handle", "user":
			attr = ui.UserName
		case "id", "internalid", "protocolid":
			attr = ui.UserID
		case "mail", "email":
			attr = ui.Email
		case "fullname", "realname":
			attr = ui.FullName
		case "firstname", "givenname":
			attr = ui.FirstName
		case "lastname", "surname":
			attr = ui.LastName
		case "phone":
			attr = ui.Phone
		}
		if len(attr) > 0 {
			return &robot.AttrRet{attr, robot.Ok}
		}
	}
	attr, ret := interfaces.GetProtocolUserAttribute(user, a)
	return &robot.AttrRet{attr, ret}
}

// messageHeard sends a typing notification
func (r *Robot) messageHeard() {
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	interfaces.MessageHeard(user, channel)
}

// GetSenderAttribute returns a AttrRet with
// - The string Attribute of the sender, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: (see above)
func (r *Robot) GetSenderAttribute(a string) *robot.AttrRet {
	c := r.getLockedContext()
	defer c.Unlock()
	a = strings.ToLower(a)
	var ui *UserInfo
	ui, _ = c.maps.user[r.User]
	switch a {
	case "name", "username", "handle", "user":
		return &robot.AttrRet{r.User, robot.Ok}
	case "id", "internalid", "protocolid":
		return &robot.AttrRet{r.ProtocolUser, robot.Ok}
	}
	if ui != nil {
		var attr string
		switch a {
		case "mail", "email":
			attr = ui.Email
		case "fullname", "realname":
			attr = ui.FullName
		case "firstname", "givenname":
			attr = ui.FirstName
		case "lastname", "surname":
			attr = ui.LastName
		case "phone":
			attr = ui.Phone
		}
		if len(attr) > 0 {
			return &robot.AttrRet{attr, robot.Ok}
		}
	}
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	attr, ret := interfaces.GetProtocolUserAttribute(user, a)
	return &robot.AttrRet{attr, ret}
}

// SendChannelMessage lets a plugin easily send a message to an arbitrary
// channel. Use Robot.Fixed().SendChannelMessage(...) for fixed-width
// font.
func (r Robot) SendChannelMessage(ch, msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in SendChannelMessage")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	c := r.getLockedContext()
	defer c.Unlock()
	var channel string
	if ci, ok := c.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return interfaces.SendProtocolChannelMessage(channel, msg, r.Format)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Note that this will fail with UserNotFound if the connector
// can't resolve usernames, or the username isn't mapped to a user ID in
// the UserRoster.
func (r Robot) SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in SendUserChannelMessage")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	c := r.getLockedContext()
	defer c.Unlock()
	var user string
	if ui, ok := c.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	var channel string
	if ci, ok := c.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return interfaces.SendProtocolUserChannelMessage(user, u, channel, msg, r.Format)
}

// SendUserMessage lets a plugin easily send a DM to a user. If a DM
// fails, an error should be returned, since DMs may be used for sending
// secret/sensitive information.
func (r Robot) SendUserMessage(u, msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in SendUserMessage")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	c := r.getLockedContext()
	defer c.Unlock()
	var user string
	if ui, ok := c.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	return interfaces.SendProtocolUserMessage(user, msg, r.Format)
}

// Reply directs a message to the user
func (r Robot) Reply(msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in Reply")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	// Support for Direct()
	if r.Channel == "" {
		return interfaces.SendProtocolUserMessage(user, msg, r.Format)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	c := r.getLockedContext()
	defer c.Unlock()
	if c != nil && c.BotUser {
		return interfaces.SendProtocolChannelMessage(r.Channel, r.User+": "+msg, r.Format)
	}
	return interfaces.SendProtocolUserChannelMessage(user, r.User, r.Channel, msg, r.Format)
}

// Say just sends a message to the user or channel
func (r Robot) Say(msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in Say")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	// Support for Direct()
	if r.Channel == "" {
		user := r.ProtocolUser
		if len(user) == 0 {
			user = r.User
		}
		return interfaces.SendProtocolUserMessage(user, msg, r.Format)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	return interfaces.SendProtocolChannelMessage(channel, msg, r.Format)
}
