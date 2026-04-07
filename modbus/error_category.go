package modbus

import "strings"

func errorCategory(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case containsAny(msg, "timeout", "i/o timeout", "deadline exceeded"):
		return "timeout"
	case containsAny(msg, "connection refused", "connection reset", "broken pipe", "no such file", "network"):
		return "connection"
	case containsAny(msg, "modbus", "exception", "crc", "framing"):
		return "protocol"
	default:
		return "other"
	}
}
