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

func TestMyPosts(t *testing.T) {
	m := getTestClient(t)
	listing, err := m.MyPosts(5)
	if err != nil {
		t.Fatalf("MyPosts() error: %v", err)
	}
	t.Logf("Got %d posts", len(listing.Posts))
	for _, p := range listing.Posts {
		t.Logf("  [r/%s] %s — %d pts", p.Subreddit, p.Title, p.Score)
	}
}

func TestMyComments(t *testing.T) {
	m := getTestClient(t)
	listing, err := m.MyComments(5)
	if err != nil {
		t.Fatalf("MyComments() error: %v", err)
	}
	t.Logf("Got %d comments", len(listing.Comments))
	for _, c := range listing.Comments {
		body := c.Body
		if len(body) > 80 {
			body = body[:80] + "..."
		}
		t.Logf("  [r/%s] %s — %d pts", c.Subreddit, body, c.Score)
	}
}

func TestUserAbout(t *testing.T) {
	m := getTestClient(t)
	me, err := m.Me()
	if err != nil {
		t.Fatalf("Me() error: %v", err)
	}
	user, err := m.UserAbout(me.Name)
	if err != nil {
		t.Fatalf("UserAbout() error: %v", err)
	}
	if user.Name == "" {
		t.Fatal("UserAbout() returned empty name")
	}
	t.Logf("User: %s, karma: %d (link: %d, comment: %d)", user.Name, user.TotalKarma, user.LinkKarma, user.CommentKarma)
}

func TestSearch(t *testing.T) {
	m := getTestClient(t)
	listing, err := m.Search("golang", 5)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	t.Logf("Got %d search results for 'golang'", len(listing.Posts))
	for _, p := range listing.Posts {
		t.Logf("  [r/%s] %s — %d pts", p.Subreddit, p.Title, p.Score)
	}
}

func TestMySubscriptions(t *testing.T) {
	m := getTestClient(t)
	listing, err := m.MySubscriptions(5)
	if err != nil {
		t.Fatalf("MySubscriptions() error: %v", err)
	}
	t.Logf("Got %d subscriptions", len(listing.Subreddits))
	for _, sr := range listing.Subreddits {
		t.Logf("  r/%s — %d subscribers", sr.Name, sr.Subscribers)
	}
}

func TestSubredditAbout(t *testing.T) {
	m := getTestClient(t)
	info, err := m.SubredditAbout("golang")
	if err != nil {
		t.Fatalf("SubredditAbout() error: %v", err)
	}
	if info.Name == "" {
		t.Fatal("SubredditAbout() returned empty name")
	}
	t.Logf("r/%s — %s (%d subscribers)", info.Name, info.Title, info.Subscribers)
}

func TestPreferences(t *testing.T) {
	m := getTestClient(t)
	prefs, err := m.Preferences()
	if err != nil {
		t.Fatalf("Preferences() error: %v", err)
	}
	if len(prefs) == 0 {
		t.Fatal("Preferences() returned empty map")
	}
	t.Logf("Got %d preference keys", len(prefs))
}

func TestTrophies(t *testing.T) {
	m := getTestClient(t)
	trophies, err := m.Trophies()
	if err != nil {
		t.Fatalf("Trophies() error: %v", err)
	}
	t.Logf("Got %d trophies", len(trophies))
	for _, tr := range trophies {
		t.Logf("  %s", tr.Name)
	}
}
