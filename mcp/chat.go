package mcp

import (
	"context"
	"time"

	reddit "github.com/teslashibe/reddit-go"
	"github.com/teslashibe/mcptool"
)

// ChatWhoAmIInput is the typed input for reddit_chat_whoami. It takes no arguments.
type ChatWhoAmIInput struct{}

func chatWhoAmI(_ context.Context, c *reddit.Client, _ ChatWhoAmIInput) (any, error) {
	return c.ChatWhoAmI()
}

// ChatRoomsInput is the typed input for reddit_chat_rooms. It takes no arguments.
type ChatRoomsInput struct{}

func chatRooms(_ context.Context, c *reddit.Client, _ ChatRoomsInput) (any, error) {
	return c.ChatRooms()
}

// ChatMessagesInput is the typed input for reddit_chat_messages.
type ChatMessagesInput struct {
	RoomID string `json:"room_id" jsonschema:"description=Matrix room ID (e.g. !abc:reddit.com),required"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=max messages to return,minimum=1,maximum=100,default=20"`
}

type chatMessageOutput struct {
	EventID   string `json:"event_id"`
	RoomID    string `json:"room_id"`
	Sender    string `json:"sender"`
	Body      string `json:"body"`
	MsgType   string `json:"msgtype,omitempty"`
	Type      string `json:"type,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

func chatMessages(_ context.Context, c *reddit.Client, in ChatMessagesInput) (any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	res, err := c.ChatMessages(in.RoomID, limit)
	if err != nil {
		return nil, err
	}
	return chatMessagePage(res.Messages, res.End, limit), nil
}

// ChatMessagesFromInput is the typed input for reddit_chat_messages_from.
type ChatMessagesFromInput struct {
	RoomID string `json:"room_id" jsonschema:"description=Matrix room ID (e.g. !abc:reddit.com),required"`
	From   string `json:"from" jsonschema:"description=pagination token returned by a previous reddit_chat_messages call (the End field),required"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=max messages to return,minimum=1,maximum=100,default=20"`
}

func chatMessagesFrom(_ context.Context, c *reddit.Client, in ChatMessagesFromInput) (any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	res, err := c.ChatMessagesFrom(in.RoomID, limit, in.From)
	if err != nil {
		return nil, err
	}
	return chatMessagePage(res.Messages, res.End, limit), nil
}

func chatMessagePage(messages []reddit.ChatMessage, cursor string, limit int) any {
	items := make([]chatMessageOutput, 0, len(messages))
	for _, msg := range messages {
		items = append(items, chatMessageOutput{
			EventID:   msg.EventID,
			RoomID:    msg.RoomID,
			Sender:    msg.Sender,
			Body:      msg.Body,
			MsgType:   msg.MsgType,
			Type:      msg.Type,
			CreatedAt: formatChatMessageTime(msg.Created),
		})
	}
	return mcptool.PageOf(items, cursor, limit)
}

func formatChatMessageTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// ChatMembersInput is the typed input for reddit_chat_members.
type ChatMembersInput struct {
	RoomID string `json:"room_id" jsonschema:"description=Matrix room ID (e.g. !abc:reddit.com),required"`
}

func chatMembers(_ context.Context, c *reddit.Client, in ChatMembersInput) (any, error) {
	return c.ChatMembers(in.RoomID)
}

// ChatSendInput is the typed input for reddit_chat_send.
type ChatSendInput struct {
	RoomID string `json:"room_id" jsonschema:"description=Matrix room ID to post in (e.g. !abc:reddit.com),required"`
	Body   string `json:"body" jsonschema:"description=plain-text message body,required"`
}

func chatSend(_ context.Context, c *reddit.Client, in ChatSendInput) (any, error) {
	eventID, err := c.ChatSend(in.RoomID, in.Body)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "room_id": in.RoomID, "event_id": eventID}, nil
}

// ChatCreateDMInput is the typed input for reddit_chat_create_dm.
type ChatCreateDMInput struct {
	MatrixUserID string `json:"matrix_user_id" jsonschema:"description=Matrix user ID of the recipient (format: @t2_xxxxx:reddit.com),required"`
}

func chatCreateDM(_ context.Context, c *reddit.Client, in ChatCreateDMInput) (any, error) {
	room, err := c.ChatCreateDM(in.MatrixUserID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "room_id": room.RoomID, "matrix_user_id": in.MatrixUserID}, nil
}

var chatTools = []mcptool.Tool{
	mcptool.Define[*reddit.Client, ChatWhoAmIInput](
		"reddit_chat_whoami",
		"Return the Matrix-side identity (user ID, device ID) of the authenticated user for Reddit chat",
		"ChatWhoAmI",
		chatWhoAmI,
	),
	mcptool.Define[*reddit.Client, ChatRoomsInput](
		"reddit_chat_rooms",
		"List the Reddit chat rooms the authenticated user has joined",
		"ChatRooms",
		chatRooms,
	),
	mcptool.Define[*reddit.Client, ChatMessagesInput](
		"reddit_chat_messages",
		"Fetch the most recent messages from a Reddit chat room (newest first)",
		"ChatMessages",
		chatMessages,
	),
	mcptool.Define[*reddit.Client, ChatMessagesFromInput](
		"reddit_chat_messages_from",
		"Fetch chat messages from a Reddit chat room starting at a pagination token (page backwards)",
		"ChatMessagesFrom",
		chatMessagesFrom,
	),
	mcptool.Define[*reddit.Client, ChatMembersInput](
		"reddit_chat_members",
		"List the members of a Reddit chat room",
		"ChatMembers",
		chatMembers,
	),
	mcptool.Define[*reddit.Client, ChatSendInput](
		"reddit_chat_send",
		"Send a plain-text message to a Reddit chat room (returns the Matrix event ID)",
		"ChatSend",
		chatSend,
	),
	mcptool.Define[*reddit.Client, ChatCreateDMInput](
		"reddit_chat_create_dm",
		"Create a direct-message chat room with another Reddit user by Matrix user ID",
		"ChatCreateDM",
		chatCreateDM,
	),
}
