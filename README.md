# reddit-messenger-go

Send and read Reddit direct messages — both legacy private messages and real-time chat.

No API keys, no OAuth app registration, zero dependencies.

## Two messaging systems, one client

Reddit has two messaging systems. This package supports both:

| System | Endpoint | Use case |
|--------|----------|----------|
| **Legacy PMs** | `oauth.reddit.com` | Traditional private messages, any user |
| **Reddit Chat** | `matrix.redditspace.com` | Real-time chat rooms (Matrix protocol) |

Both authenticate with the same `token_v2` cookie from your browser.

## Install

```bash
go get github.com/teslashibe/reddit-messenger-go
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

    redditmessenger "github.com/teslashibe/reddit-messenger-go"
)

func main() {
    m := redditmessenger.New(&redditmessenger.Options{
        Token: "your_token_v2_value",
    })

    // Verify auth
    me, err := m.Me()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Logged in as: %s\n", me.Name)

    // Read inbox
    inbox, _ := m.Inbox(5)
    for _, msg := range inbox.Messages {
        fmt.Printf("[%s] %s: %s\n", msg.Type, msg.Author, msg.Subject)
    }

    // Send a PM
    _ = m.Compose("username", "Hello", "Message body here")

    // List chat rooms
    rooms, _ := m.ChatRooms()
    for _, r := range rooms {
        fmt.Println(r.RoomID)
    }

    // Read chat messages
    msgs, _ := m.ChatMessages(rooms[0].RoomID, 10)
    for _, msg := range msgs.Messages {
        fmt.Printf("%s: %s\n", msg.Sender, msg.Body)
    }

    // Send a chat message
    eventID, _ := m.ChatSend(rooms[0].RoomID, "Hello from Go!")
    fmt.Printf("Sent: %s\n", eventID)
}
```

## API

### Legacy Private Messages

```go
m.Me()                              // Verify auth, get username
m.Inbox(limit)                      // All inbox items (PMs + replies)
m.Messages(limit)                   // PMs only
m.Sent(limit)                       // Sent messages
m.Unread(limit)                     // Unread items
m.Compose(to, subject, body)        // Send a PM
m.MarkRead(fullnames...)            // Mark as read
m.MarkUnread(fullnames...)          // Mark as unread
```

Pagination:

```go
page1, _ := m.Inbox(25)
page2, _ := m.InboxAfter(25, page1.After)
```

### Reddit Chat (Matrix)

```go
m.ChatWhoAmI()                      // Matrix identity
m.ChatRooms()                       // List joined rooms
m.ChatMessages(roomID, limit)       // Read messages (newest first)
m.ChatMessagesFrom(roomID, limit, token) // Paginate backwards
m.ChatMembers(roomID)               // Room members
m.ChatSend(roomID, text)            // Send message
m.ChatCreateDM(matrixUserID)        // Create DM room
```

Matrix user IDs are `@t2_xxxxx:reddit.com` format.

## CLI

```bash
go install github.com/teslashibe/reddit-messenger-go/cmd/reddit-msg@latest
export REDDIT_TOKEN="your_token_v2_value"
```

```bash
# Auth
reddit-msg me

# Legacy PMs
reddit-msg inbox --limit 10
reddit-msg messages --limit 5
reddit-msg unread
reddit-msg send --to username --subject "Hi" --body "Hello there"

# Chat
reddit-msg chat whoami
reddit-msg chat rooms
reddit-msg chat messages --room '!roomid:reddit.com' --limit 20
reddit-msg chat send --room '!roomid:reddit.com' --body "Hello!"
reddit-msg chat members --room '!roomid:reddit.com'

# JSON output
reddit-msg inbox --limit 5 --json
reddit-msg chat messages --room '!roomid:reddit.com' --json
```

## Design

- **Zero dependencies** — stdlib only
- **Rate limiting** — leaky-bucket pattern respecting Reddit's `X-Ratelimit-*` headers
- **Same auth model** as [reddit-scraper](https://github.com/teslashibe/reddit-scraper) — `token_v2` cookie

## License

MIT
