package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/teslashibe/reddit-go"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	token := os.Getenv("REDDIT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: REDDIT_TOKEN environment variable required (token_v2 cookie value)")
		os.Exit(1)
	}

	m := reddit.New(&reddit.Options{Token: token})

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "me":
		cmdMe(m)
	case "inbox":
		cmdInbox(m, args)
	case "messages":
		cmdMessages(m, args)
	case "sent":
		cmdSent(m, args)
	case "unread":
		cmdUnread(m, args)
	case "send":
		cmdSend(m, args)
	case "chat":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: reddit-msg chat <rooms|messages|send|members|whoami>")
			os.Exit(1)
		}
		subcmd := args[0]
		subargs := args[1:]
		switch subcmd {
		case "whoami":
			cmdChatWhoAmI(m)
		case "rooms":
			cmdChatRooms(m)
		case "messages":
			cmdChatMessages(m, subargs)
		case "send":
			cmdChatSend(m, subargs)
		case "members":
			cmdChatMembers(m, subargs)
		default:
			fmt.Fprintf(os.Stderr, "unknown chat subcommand: %s\n", subcmd)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `reddit-msg — Reddit DM & Chat CLI

Commands:
  me                         Show authenticated user
  inbox    [--limit N]       Read inbox
  messages [--limit N]       Read private messages only
  sent     [--limit N]       Read sent messages
  unread   [--limit N]       Read unread items
  send     --to USER --subject SUBJ --body TEXT
                             Send a private message

  chat whoami                Show chat identity
  chat rooms                 List chat rooms
  chat messages --room ID [--limit N]
                             Read chat room messages
  chat send --room ID --body TEXT
                             Send chat message
  chat members --room ID     List room members

Environment:
  REDDIT_TOKEN               token_v2 cookie value (required)`)
}

func cmdMe(m *reddit.Client) {
	id, err := m.Me()
	fatal(err)
	fmt.Printf("User: %s (ID: %s)\n", id.Name, id.ID)
}

func cmdInbox(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("inbox", flag.ExitOnError)
	limit := fs.Int("limit", 10, "number of messages")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	listing, err := m.Inbox(*limit)
	fatal(err)
	printMessages(listing, *jsonOut)
}

func cmdMessages(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("messages", flag.ExitOnError)
	limit := fs.Int("limit", 10, "number of messages")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	listing, err := m.Messages(*limit)
	fatal(err)
	printMessages(listing, *jsonOut)
}

func cmdSent(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("sent", flag.ExitOnError)
	limit := fs.Int("limit", 10, "number of messages")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	listing, err := m.Sent(*limit)
	fatal(err)
	printMessages(listing, *jsonOut)
}

func cmdUnread(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("unread", flag.ExitOnError)
	limit := fs.Int("limit", 10, "number of messages")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	listing, err := m.Unread(*limit)
	fatal(err)
	printMessages(listing, *jsonOut)
}

func cmdSend(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	to := fs.String("to", "", "recipient username")
	subject := fs.String("subject", "", "message subject")
	body := fs.String("body", "", "message body")
	_ = fs.Parse(args)

	if *to == "" || *subject == "" || *body == "" {
		fmt.Fprintln(os.Stderr, "error: --to, --subject, and --body are required")
		os.Exit(1)
	}

	err := m.Compose(*to, *subject, *body)
	fatal(err)
	fmt.Printf("Message sent to u/%s\n", *to)
}

func cmdChatWhoAmI(m *reddit.Client) {
	id, err := m.ChatWhoAmI()
	fatal(err)
	fmt.Printf("Matrix ID: %s\nDevice: %s\n", id.UserID, id.DeviceID)
}

func cmdChatRooms(m *reddit.Client) {
	rooms, err := m.ChatRooms()
	fatal(err)
	fmt.Printf("Found %d rooms:\n", len(rooms))
	for i, r := range rooms {
		fmt.Printf("  %d. %s\n", i+1, r.RoomID)
	}
}

func cmdChatMessages(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("chat messages", flag.ExitOnError)
	room := fs.String("room", "", "room ID")
	limit := fs.Int("limit", 10, "number of messages")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	if *room == "" {
		fmt.Fprintln(os.Stderr, "error: --room is required")
		os.Exit(1)
	}

	listing, err := m.ChatMessages(*room, *limit)
	fatal(err)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(listing.Messages))
		return
	}

	fmt.Printf("Messages in room (newest first):\n\n")
	for _, msg := range listing.Messages {
		ts := msg.Created.Format(time.RFC3339)
		body := msg.Body
		if len(body) > 120 {
			body = body[:120] + "..."
		}
		fmt.Printf("[%s] %s:\n  %s\n\n", ts, msg.Sender, body)
	}
}

func cmdChatSend(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("chat send", flag.ExitOnError)
	room := fs.String("room", "", "room ID")
	body := fs.String("body", "", "message text")
	_ = fs.Parse(args)

	if *room == "" || *body == "" {
		fmt.Fprintln(os.Stderr, "error: --room and --body are required")
		os.Exit(1)
	}

	eventID, err := m.ChatSend(*room, *body)
	fatal(err)
	fmt.Printf("Sent (event: %s)\n", eventID)
}

func cmdChatMembers(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("chat members", flag.ExitOnError)
	room := fs.String("room", "", "room ID")
	_ = fs.Parse(args)

	if *room == "" {
		fmt.Fprintln(os.Stderr, "error: --room is required")
		os.Exit(1)
	}

	members, err := m.ChatMembers(*room)
	fatal(err)

	fmt.Printf("Members (%d):\n", len(members))
	for _, mb := range members {
		name := mb.DisplayName
		if mb.Username != "" {
			name = mb.Username
		}
		fmt.Printf("  %s (%s) — %s\n", name, mb.UserID, mb.Membership)
	}
}

func printMessages(listing *reddit.MessageListing, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(listing.Messages))
		return
	}

	if len(listing.Messages) == 0 {
		fmt.Println("No messages.")
		return
	}

	for _, msg := range listing.Messages {
		ts := msg.Created.Format(time.RFC3339)
		typ := "PM"
		if msg.WasComment {
			typ = "comment"
		}

		body := msg.Body
		if len(body) > 120 {
			body = body[:120] + "..."
		}
		body = strings.ReplaceAll(body, "\n", "\n  ")

		fmt.Printf("[%s] [%s] from u/%s — %s\n", ts, typ, msg.Author, msg.Subject)
		fmt.Printf("  %s\n\n", body)
	}

	if listing.After != "" {
		fmt.Printf("(more available, after=%s)\n", listing.After)
	}
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
