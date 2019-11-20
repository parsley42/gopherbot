package robot

import "regexp"

var IDRegex = regexp.MustCompile(`^<(.*)>$`)

// ExtractID is a utility function to check a user/channel string against
// the pattern '<internalID>' and if it matches return the internalID,true;
// otherwise return the unmodified string,false.
func ExtractID(u string) (string, bool) {
	matches := IDRegex.FindStringSubmatch(u)
	if len(matches) > 0 {
		return matches[1], true
	}
	return u, false
}
