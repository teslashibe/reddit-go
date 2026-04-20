# reddit-go

Full-surface Reddit API client — messaging, chat, posts, comments, voting, search, subreddits, and account.

No API keys, no OAuth app registration, zero dependencies.

## Install

```bash
go get github.com/teslashibe/reddit-go
```

## Get your token

1. Log in to reddit.com in your browser
2. Open DevTools → Application → Cookies → reddit.com
3. Copy the value of `token_v2`

## Quick start

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

    // Identity
    me, err := c.Me()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Logged in as: %s\n", me.Name)

    // Read inbox & send PMs
    inbox, _ := c.Inbox(5)
    for _, msg := range inbox.Messages {
        fmt.Printf("[%s] %s: %s\n", msg.Type, msg.Author, msg.Subject)
    }
    _ = c.Compose("username", "Hello", "Message body here")

    // Posts & comments
    posts, _ := c.MyPosts(10)
    for _, p := range posts.Posts {
        fmt.Printf("%s — %d pts\n", p.Title, p.Score)
    }
    _ = c.Submit("test", "Title", "Body")
    _, _ = c.Reply("t3_postid", "comment text")

    // Chat (Matrix)
    rooms, _ := c.ChatRooms()
    _, _ = c.ChatSend(rooms[0].RoomID, "Hello from Go!")

    // Search, vote, subscribe
    results, _ := c.Search("golang", 10)
    fmt.Printf("Found %d results\n", len(results.Posts))
    _ = c.Upvote("t3_postid")
    _ = c.Subscribe("golang")
}
```

## API

### Identity & Account

```go
c.Me()                              // Authenticated user identity
c.Preferences()                     // Account preferences (raw map)
c.Trophies()                        // Trophy list
c.Friends()                         // Friend list
```

### Legacy Private Messages

```go
c.Inbox(limit)                      // All inbox items (PMs + replies)
c.InboxAfter(limit, after)          // Next page of inbox
c.Messages(limit)                   // PMs only
c.MessagesAfter(limit, after)       // Next page of PMs
c.Sent(limit)                       // Sent messages
c.SentAfter(limit, after)           // Next page of sent
c.Unread(limit)                     // Unread items
c.CommentReplies(limit)             // Replies to your comments
c.PostReplies(limit)                // Replies to your posts
c.Compose(to, subject, body)        // Send a PM
c.MarkRead(fullnames...)            // Mark as read
c.MarkUnread(fullnames...)          // Mark as unread
```

### Posts & Comments

```go
c.MyPosts(limit)                    // Your submitted posts
c.MyComments(limit)                 // Your comments
c.MyOverview(limit)                 // Mixed post + comment feed
c.Saved(limit)                      // Saved items
c.Upvoted(limit)                    // Upvoted posts
c.Submit(subreddit, title, body)    // Create a self-post
c.SubmitLink(subreddit, title, url) // Create a link post
c.Reply(parentID, body)             // Comment on a post/comment
c.Edit(fullname, newText)           // Edit a self-post or comment
c.Delete(fullname)                  // Delete a post or comment
```

### Voting & Actions

```go
c.Upvote(fullname)                  // Upvote a post or comment
c.Downvote(fullname)                // Downvote
c.Unvote(fullname)                  // Remove vote
c.Save(fullname)                    // Bookmark
c.Unsave(fullname)                  // Remove bookmark
c.Hide(fullname)                    // Hide from feed
c.Unhide(fullname)                  // Unhide
c.Report(fullname, reason)          // Report
```

### Search

```go
c.Search(query, limit)                          // Search all of Reddit
c.SearchInSubreddit(subreddit, query, limit)     // Search within a subreddit
```

### Users

```go
c.UserAbout(username)               // Public info about any user
```

### Subreddits

```go
c.MySubscriptions(limit)            // Your subscribed subreddits
c.SubredditAbout(name)              // Subreddit metadata
c.Subscribe(subreddit)              // Subscribe
c.Unsubscribe(subreddit)            // Unsubscribe
```

### Reddit Chat (Matrix)

```go
c.ChatWhoAmI()                      // Matrix identity
c.ChatRooms()                       // List joined rooms
c.ChatMessages(roomID, limit)       // Read messages (newest first)
c.ChatMessagesFrom(roomID, limit, token) // Paginate backwards
c.ChatMembers(roomID)               // Room members
c.ChatSend(roomID, text)            // Send message
c.ChatCreateDM(matrixUserID)        // Create DM room
```

Matrix user IDs are `@t2_xxxxx:reddit.com` format.

## CLI

```bash
go install github.com/teslashibe/reddit-go/cmd/reddit-msg@latest
export REDDIT_TOKEN="your_token_v2_value"
```

```bash
# Identity
reddit-msg me

# Legacy PMs
reddit-msg inbox [--limit N]
reddit-msg messages [--limit N]
reddit-msg sent [--limit N]
reddit-msg unread [--limit N]
reddit-msg send --to USER --subject SUBJ --body TEXT

# Posts & comments
reddit-msg posts [--limit N]
reddit-msg comments [--limit N]

# Search
reddit-msg search --query Q [--limit N] [--subreddit SR]

# Subreddits
reddit-msg subs [--limit N]
reddit-msg subreddit --name NAME

# Users
reddit-msg user --name USERNAME

# Chat
reddit-msg chat whoami
reddit-msg chat rooms
reddit-msg chat messages --room ID [--limit N]
reddit-msg chat send --room ID --body TEXT
reddit-msg chat members --room ID

# JSON output (add --json to most commands)
reddit-msg inbox --limit 5 --json
```

## Design

- **Zero dependencies** — stdlib only
- **Rate limiting** — leaky-bucket pattern respecting Reddit's `X-Ratelimit-*` headers
- **Same auth model** as [reddit-scraper](https://github.com/teslashibe/reddit-scraper) — `token_v2` cookie

## License

MIT
