// Package reddit is a full-surface Reddit API client — messaging,
// posting, commenting, voting, search, subscriptions, and chat.
//
// No API keys, no OAuth app registration, zero dependencies.
// Authenticate with a token_v2 cookie from your logged-in browser session.
//
// # Quick start
//
//	c := reddit.New(&reddit.Options{
//		Token: "your_token_v2_value",
//	})
//
//	// Identity
//	me, _ := c.Me()
//
//	// Read inbox & send PMs
//	inbox, _ := c.Inbox(10)
//	_ = c.Compose("username", "subject", "message body")
//
//	// Posts & comments
//	posts, _ := c.MyPosts(10)
//	_ = c.Submit("subreddit", "title", "body")
//	_ = c.Reply("t3_postid", "comment text")
//
//	// Chat (Matrix)
//	rooms, _ := c.ChatRooms()
//	_ = c.ChatSend(roomID, "hello!")
//
//	// Search, vote, subscribe
//	results, _ := c.Search("query", 10)
//	_ = c.Upvote("t3_postid")
//	_ = c.Subscribe("golang")
package reddit
