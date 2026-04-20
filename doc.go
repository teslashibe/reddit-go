// Package redditmessenger sends and reads Reddit direct messages —
// both legacy private messages (via oauth.reddit.com) and real-time
// chat (via Reddit's Matrix-based chat at matrix.redditspace.com).
//
// No API keys, no OAuth app registration, zero dependencies.
// Authenticate with a token_v2 cookie from your logged-in browser session.
//
// # Quick start
//
//	m := redditmessenger.New(&redditmessenger.Options{
//		Token: "your_token_v2_value",
//	})
//
//	// Read inbox
//	msgs, err := m.Inbox(10)
//
//	// Send a PM
//	err = m.Compose("username", "subject", "message body")
//
//	// List chat rooms
//	rooms, err := m.ChatRooms()
//
//	// Read chat messages
//	msgs, err := m.ChatMessages(roomID, 20)
//
//	// Send a chat message
//	err = m.ChatSend(roomID, "hello!")
package redditmessenger
