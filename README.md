# reddit-go

**Full-surface Reddit API client for Go** — messaging, chat, posts, comments, voting, search, subreddits, and account management.

No API keys. No OAuth app registration. Zero dependencies. Just a browser cookie.

---

## Why

Reddit has two official API paths: register an OAuth app (rate-limited, approval-gated) or scrape HTML. Both are painful. Your browser already has a fully-authenticated session with every scope unlocked. This package uses that session token directly — one cookie, full access, zero ceremony.

Covers **every user-facing Reddit surface**:

| Surface | Read | Write |
|---------|------|-------|
| Private messages | Inbox, sent, unread | Compose, mark read |
| Chat (Matrix) | Rooms, messages, members | Send, create DM |
| Posts | My posts, any user's posts | Submit, edit, delete |
| Comments | My comments, replies | Reply, edit, delete |
| Voting | — | Upvote, downvote, unvote |
| Search | Posts, subreddit-scoped | — |
| Subreddits | About, my subscriptions | Subscribe, unsubscribe |
| Account | Preferences, trophies, friends | — |
| Actions | — | Save, hide, report |

---

## Install

```bash
go get github.com/teslashibe/reddit-go
```

**Requires:** Go 1.21+, zero external dependencies.

---

## Authentication

Grab your `token_v2` cookie from any logged-in Reddit session:

1. Open **reddit.com** in your browser (must be logged in)
2. DevTools → **Application** → **Cookies** → `reddit.com`
3. Find `token_v2` → copy the value

```go
c := reddit.New(&reddit.Options{
    Token: "eyJhbGciOiJS...", // your token_v2 value
})
```

> The token is a JWT with full OAuth scopes. No app registration needed.

---

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/teslashibe/reddit-go"
)

func main() {
    c := reddit.New(&reddit.Options{
        Token: "your_token_v2_value",
    })

    // Who am I?
    me, err := c.Me()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Logged in as u/%s\n", me.Name)

    // Read inbox
    inbox, _ := c.Inbox(5)
    for _, msg := range inbox.Messages {
        fmt.Printf("  [%s] u/%s: %s\n", msg.Type, msg.Author, msg.Subject)
    }

    // My posts
    posts, _ := c.MyPosts(10)
    for _, p := range posts.Posts {
        fmt.Printf("  r/%s — %s (%d pts)\n", p.Subreddit, p.Title, p.Score)
    }

    // Send a chat message
    rooms, _ := c.ChatRooms()
    if len(rooms) > 0 {
        c.ChatSend(rooms[0].RoomID, "Hello from reddit-go!")
    }

    // Search
    results, _ := c.Search("golang", 5)
    fmt.Printf("Found %d results\n", len(results.Posts))
}
```

---

## API Reference

### Identity & Account

```go
c.Me()              // → *Identity        Your username and ID
c.Preferences()     // → map[string]any   All 87 account preferences
c.Trophies()        // → []Trophy         Awards (Five-Year Club, etc.)
c.Friends()         // → []Friend         Friend list
```

### Messaging (Legacy PMs)

```go
// Read
c.Inbox(limit)                  // All inbox (PMs + comment replies)
c.Messages(limit)               // PMs only
c.Sent(limit)                   // Sent messages
c.Unread(limit)                 // Unread items
c.CommentReplies(limit)         // Replies to your comments
c.PostReplies(limit)            // Replies to your posts

// Pagination
c.InboxAfter(limit, cursor)     // Next page
c.MessagesAfter(limit, cursor)
c.SentAfter(limit, cursor)

// Write
c.Compose(to, subject, body)    // Send PM to any user
c.MarkRead(fullnames...)        // Mark as read
c.MarkUnread(fullnames...)      // Mark as unread
```

### Chat (Reddit's Matrix-based real-time chat)

```go
c.ChatWhoAmI()                       // → *ChatIdentity    Matrix user ID
c.ChatRooms()                        // → []ChatRoom       Joined rooms
c.ChatMessages(roomID, limit)        // → *ChatMessageListing
c.ChatMessagesFrom(roomID, limit, t) // Paginate backwards
c.ChatMembers(roomID)                // → []ChatMember
c.ChatSend(roomID, text)             // → eventID, error
c.ChatCreateDM(matrixUserID)         // → *ChatRoom        New DM room
```

> Matrix user IDs follow the format `@t2_xxxxx:reddit.com`

### Posts & Comments

```go
// Read your content
c.MyPosts(limit)                     // Your submitted posts
c.MyComments(limit)                  // Your comments
c.MyOverview(limit)                  // Posts + comments interleaved
c.Saved(limit)                       // Bookmarked items
c.Upvoted(limit)                     // Posts you upvoted

// Create
c.Submit(subreddit, title, body)     // → *Post   Self-post
c.SubmitLink(subreddit, title, url)  // → *Post   Link post
c.Reply(parentID, body)              // → commentID, error

// Modify
c.Edit(fullname, newText)            // Edit your post or comment
c.Delete(fullname)                   // Delete your post or comment
```

### Voting & Actions

```go
c.Upvote(fullname)                   // +1
c.Downvote(fullname)                 // -1
c.Unvote(fullname)                   // Remove vote
c.Save(fullname)                     // Bookmark
c.Unsave(fullname)                   // Remove bookmark
c.Hide(fullname)                     // Hide from feed
c.Unhide(fullname)                   // Show again
c.Report(fullname, reason)           // Report to mods
```

### Search

```go
c.Search(query, limit)                         // Search all of Reddit
c.SearchInSubreddit(subreddit, query, limit)   // Scoped to one sub
```

### Users & Subreddits

```go
c.UserAbout(username)                // → *User           Any user's public profile
c.SubredditAbout(name)               // → *SubredditInfo  Subreddit metadata
c.MySubscriptions(limit)             // → *SubredditListing
c.Subscribe(subreddit)               // Join a subreddit
c.Unsubscribe(subreddit)             // Leave a subreddit
```

---

## CLI

```bash
go install github.com/teslashibe/reddit-go/cmd/reddit-msg@latest
export REDDIT_TOKEN="your_token_v2_value"
```

```bash
# Identity
reddit-msg me

# Messaging
reddit-msg inbox --limit 10
reddit-msg messages --limit 5
reddit-msg unread
reddit-msg send --to spez --subject "Hello" --body "Hi from reddit-go"

# Posts & Comments
reddit-msg posts --limit 10
reddit-msg comments --limit 10

# Search
reddit-msg search --query "golang generics" --limit 5
reddit-msg search --query "performance" --subreddit golang --limit 5

# Subreddits & Users
reddit-msg subs --limit 10
reddit-msg subreddit --name golang
reddit-msg user --name spez

# Chat
reddit-msg chat rooms
reddit-msg chat messages --room '!roomid:reddit.com' --limit 20
reddit-msg chat send --room '!roomid:reddit.com' --body "Hello!"
reddit-msg chat members --room '!roomid:reddit.com'
reddit-msg chat whoami

# JSON output (works with most commands)
reddit-msg inbox --limit 5 --json
reddit-msg posts --json
```

---

## Design

| Principle | Detail |
|-----------|--------|
| **Zero dependencies** | stdlib only — `net/http`, `encoding/json`, `net/url` |
| **Rate limiting** | Leaky-bucket reservation pattern respecting `X-Ratelimit-*` headers |
| **Dual transport** | `oauth.reddit.com` for REST API + `matrix.redditspace.com` for chat |
| **Auth model** | `token_v2` browser cookie as Bearer token — same approach as [reddit-scraper](https://github.com/teslashibe/reddit-scraper) |

### Architecture

```
reddit.New(opts)
    │
    ├── oauth.reddit.com        ← REST API (messages, posts, voting, search, ...)
    │     └── Bearer token_v2
    │
    └── matrix.redditspace.com  ← Matrix protocol (real-time chat)
          └── Bearer token_v2
```

---

## Related

- **[reddit-scraper](https://github.com/teslashibe/reddit-scraper)** — Scrape posts and full comment trees from any public subreddit (unauthenticated)

---

## License

MIT
