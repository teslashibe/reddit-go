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
	case "posts":
		cmdPosts(m, args)
	case "comments":
		cmdComments(m, args)
	case "search":
		cmdSearch(m, args)
	case "subs":
		cmdSubs(m, args)
	case "user":
		cmdUser(m, args)
	case "subreddit":
		cmdSubreddit(m, args)
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
	fmt.Fprintln(os.Stderr, `reddit-msg — Reddit CLI

Commands:
  me                         Show authenticated user
  inbox    [--limit N]       Read inbox
  messages [--limit N]       Read private messages only
  sent     [--limit N]       Read sent messages
  unread   [--limit N]       Read unread items
  send     --to USER --subject SUBJ --body TEXT
                             Send a private message
  posts    [--limit N]       Show my posts
  comments [--limit N]       Show my comments
  search   --query Q [--limit N] [--subreddit SR]
                             Search Reddit
  subs     [--limit N]       Show my subscriptions
  user     --name USERNAME   Show user info
  subreddit --name NAME      Show subreddit info

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

func cmdPosts(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("posts", flag.ExitOnError)
	limit := fs.Int("limit", 10, "number of posts")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	listing, err := m.MyPosts(*limit)
	fatal(err)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(listing.Posts))
		return
	}

	if len(listing.Posts) == 0 {
		fmt.Println("No posts.")
		return
	}

	for _, p := range listing.Posts {
		ts := p.Created.Format(time.RFC3339)
		fmt.Printf("[%s] [r/%s] %s — %d pts, %d comments\n", ts, p.Subreddit, p.Title, p.Score, p.NumComments)
		if p.SelfText != "" {
			body := p.SelfText
			if len(body) > 120 {
				body = body[:120] + "..."
			}
			body = strings.ReplaceAll(body, "\n", "\n  ")
			fmt.Printf("  %s\n", body)
		}
		fmt.Println()
	}

	if listing.After != "" {
		fmt.Printf("(more available, after=%s)\n", listing.After)
	}
}

func cmdComments(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("comments", flag.ExitOnError)
	limit := fs.Int("limit", 10, "number of comments")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	listing, err := m.MyComments(*limit)
	fatal(err)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(listing.Comments))
		return
	}

	if len(listing.Comments) == 0 {
		fmt.Println("No comments.")
		return
	}

	for _, c := range listing.Comments {
		ts := c.Created.Format(time.RFC3339)
		body := c.Body
		if len(body) > 120 {
			body = body[:120] + "..."
		}
		body = strings.ReplaceAll(body, "\n", "\n  ")
		fmt.Printf("[%s] [r/%s] %d pts — on: %s\n", ts, c.Subreddit, c.Score, c.LinkTitle)
		fmt.Printf("  %s\n\n", body)
	}

	if listing.After != "" {
		fmt.Printf("(more available, after=%s)\n", listing.After)
	}
}

func cmdSearch(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	query := fs.String("query", "", "search query")
	limit := fs.Int("limit", 10, "number of results")
	subreddit := fs.String("subreddit", "", "restrict to subreddit")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	if *query == "" {
		fmt.Fprintln(os.Stderr, "error: --query is required")
		os.Exit(1)
	}

	var listing *reddit.PostListing
	var err error
	if *subreddit != "" {
		listing, err = m.SearchInSubreddit(*subreddit, *query, *limit)
	} else {
		listing, err = m.Search(*query, *limit)
	}
	fatal(err)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(listing.Posts))
		return
	}

	if len(listing.Posts) == 0 {
		fmt.Println("No results.")
		return
	}

	for _, p := range listing.Posts {
		ts := p.Created.Format(time.RFC3339)
		fmt.Printf("[%s] [r/%s] %s — %d pts, %d comments\n", ts, p.Subreddit, p.Title, p.Score, p.NumComments)
		fmt.Printf("  %s\n\n", p.URL)
	}

	if listing.After != "" {
		fmt.Printf("(more available, after=%s)\n", listing.After)
	}
}

func cmdSubs(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("subs", flag.ExitOnError)
	limit := fs.Int("limit", 25, "number of subreddits")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	listing, err := m.MySubscriptions(*limit)
	fatal(err)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(listing.Subreddits))
		return
	}

	if len(listing.Subreddits) == 0 {
		fmt.Println("No subscriptions.")
		return
	}

	for _, sr := range listing.Subreddits {
		fmt.Printf("r/%-30s %d subscribers\n", sr.Name, sr.Subscribers)
	}

	if listing.After != "" {
		fmt.Printf("\n(more available, after=%s)\n", listing.After)
	}
}

func cmdUser(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("user", flag.ExitOnError)
	name := fs.String("name", "", "username")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: --name is required")
		os.Exit(1)
	}

	user, err := m.UserAbout(*name)
	fatal(err)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(user))
		return
	}

	fmt.Printf("User: u/%s\n", user.Name)
	fmt.Printf("  ID:            %s\n", user.ID)
	fmt.Printf("  Total Karma:   %d\n", user.TotalKarma)
	fmt.Printf("  Link Karma:    %d\n", user.LinkKarma)
	fmt.Printf("  Comment Karma: %d\n", user.CommentKarma)
	fmt.Printf("  Created:       %s\n", user.Created.Format(time.RFC3339))
	fmt.Printf("  Verified:      %v\n", user.Verified)
	fmt.Printf("  Gold:          %v\n", user.IsGold)
	fmt.Printf("  Mod:           %v\n", user.IsMod)
}

func cmdSubreddit(m *reddit.Client, args []string) {
	fs := flag.NewFlagSet("subreddit", flag.ExitOnError)
	name := fs.String("name", "", "subreddit name")
	jsonOut := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: --name is required")
		os.Exit(1)
	}

	info, err := m.SubredditAbout(*name)
	fatal(err)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatal(enc.Encode(info))
		return
	}

	fmt.Printf("Subreddit: r/%s\n", info.Name)
	fmt.Printf("  Title:       %s\n", info.Title)
	fmt.Printf("  Subscribers: %d\n", info.Subscribers)
	fmt.Printf("  Active:      %d\n", info.ActiveUsers)
	fmt.Printf("  Type:        %s\n", info.SubredditType)
	fmt.Printf("  NSFW:        %v\n", info.Over18)
	fmt.Printf("  Created:     %s\n", info.Created.Format(time.RFC3339))
	if info.Description != "" {
		desc := info.Description
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		fmt.Printf("  Description: %s\n", desc)
	}
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
