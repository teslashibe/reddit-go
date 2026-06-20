package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	reddit "github.com/teslashibe/reddit-go"
)

func TestChatMessagePageIsAgentReadable(t *testing.T) {
	msg := reddit.ChatMessage{
		EventID: "event-1",
		RoomID:  "!room:reddit.com",
		Sender:  "@t2_sender:reddit.com",
		Body:    "hello",
		MsgType: "m.text",
		Type:    "m.room.message",
		Created: time.Date(2026, 6, 20, 16, 27, 56, 0, time.UTC),
	}

	buf, err := json.Marshal(chatMessagePage([]reddit.ChatMessage{msg}, "cursor", 2))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"sender":"@t2_sender:reddit.com"`,
		`"body":"hello"`,
		`"created_at":"2026-06-20T16:27:56Z"`,
		`"next_cursor":"cursor"`,
	} {
		if !strings.Contains(string(buf), want) {
			t.Fatalf("chat message JSON missing %s: %s", want, buf)
		}
	}
}
