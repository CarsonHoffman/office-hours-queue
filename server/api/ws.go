package api

import "github.com/segmentio/ksuid"

type WSMessage struct {
	Event string      `json:"e"`
	Data  interface{} `json:"d"`
}

func WS(event string, data interface{}) *WSMessage {
	return &WSMessage{
		Event: event,
		Data:  data,
	}
}

const separator = "🦞"

func prefix(s string) string {
	return separator + s
}

func surround(s string) string {
	return separator + s + separator
}

func QueueTopicGeneric(queue ksuid.KSUID) string {
	return "queue" + prefix(queue.String())
}

func QueueTopicNonPrivileged(queue ksuid.KSUID) string {
	return "queue" + surround(queue.String()) + "non_privileged"
}

func QueueTopicAdmin(queue ksuid.KSUID) string {
	return "queue" + surround(queue.String()) + "admin"
}

func QueueTopicEmail(queue ksuid.KSUID, email string) string {
	return "queue" + surround(queue.String()) + "user" + prefix(email)
}
