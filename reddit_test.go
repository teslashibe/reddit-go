package reddit_test

import (
	"os"
	"testing"

	"github.com/teslashibe/reddit-go"
)

func getTestClient(t *testing.T) *reddit.Client {
	t.Helper()
	token := os.Getenv("REDDIT_TOKEN")
	if token == "" {
		t.Skip("REDDIT_TOKEN not set; skipping integration test")
	}
	return reddit.New(&reddit.Options{Token: token})
}

func TestMe(t *testing.T) {
	m := getTestClient(t)
	id, err := m.Me()
	if err != nil {
		t.Fatalf("Me() error: %v", err)
	}
	if id.Name == "" {
		t.Fatal("Me() returned empty name")
	}
	t.Logf("Authenticated as: %s (ID: %s)", id.Name, id.ID)
}

func TestInbox(t *testing.T) {
	m := getTestClient(t)
	listing, err := m.Inbox(3)
	if err != nil {
		t.Fatalf("Inbox() error: %v", err)
	}
	t.Logf("Got %d inbox items", len(listing.Messages))
	for _, msg := range listing.Messages {
		t.Logf("  [%s] %s → %s: %s", msg.Type, msg.Author, msg.Dest, msg.Subject)
	}
}

func TestMessages(t *testing.T) {
	m := getTestClient(t)
	listing, err := m.Messages(3)
	if err != nil {
		t.Fatalf("Messages() error: %v", err)
	}
	t.Logf("Got %d PMs", len(listing.Messages))
}

func TestUnread(t *testing.T) {
	m := getTestClient(t)
	listing, err := m.Unread(3)
	if err != nil {
		t.Fatalf("Unread() error: %v", err)
	}
	t.Logf("Got %d unread items", len(listing.Messages))
}

func TestChatWhoAmI(t *testing.T) {
	m := getTestClient(t)
	id, err := m.ChatWhoAmI()
	if err != nil {
		t.Fatalf("ChatWhoAmI() error: %v", err)
	}
	if id.UserID == "" {
		t.Fatal("ChatWhoAmI() returned empty user_id")
	}
	t.Logf("Matrix ID: %s, Device: %s", id.UserID, id.DeviceID)
}

func TestChatRooms(t *testing.T) {
	m := getTestClient(t)
	rooms, err := m.ChatRooms()
	if err != nil {
		t.Fatalf("ChatRooms() error: %v", err)
	}
	t.Logf("Got %d rooms", len(rooms))
	for _, r := range rooms {
		t.Logf("  %s", r.RoomID)
	}
}

func TestChatMessages(t *testing.T) {
	m := getTestClient(t)
	rooms, err := m.ChatRooms()
	if err != nil {
		t.Fatalf("ChatRooms() error: %v", err)
	}
	if len(rooms) == 0 {
		t.Skip("no chat rooms to read")
	}

	listing, err := m.ChatMessages(rooms[0].RoomID, 3)
	if err != nil {
		t.Fatalf("ChatMessages() error: %v", err)
	}
	t.Logf("Got %d messages from room %s", len(listing.Messages), rooms[0].RoomID)
	for _, msg := range listing.Messages {
		body := msg.Body
		if len(body) > 80 {
			body = body[:80] + "..."
		}
		t.Logf("  %s: %s", msg.Sender, body)
	}
}

func TestChatMembers(t *testing.T) {
	m := getTestClient(t)
	rooms, err := m.ChatRooms()
	if err != nil {
		t.Fatalf("ChatRooms() error: %v", err)
	}
	if len(rooms) == 0 {
		t.Skip("no chat rooms")
	}

	members, err := m.ChatMembers(rooms[0].RoomID)
	if err != nil {
		t.Fatalf("ChatMembers() error: %v", err)
	}
	t.Logf("Got %d members", len(members))
	for _, mb := range members {
		t.Logf("  %s (%s)", mb.DisplayName, mb.UserID)
	}
}
