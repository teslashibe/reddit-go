package reddit_test

import (
	"encoding/json"
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

// getTestClientWithCookies returns a client with the full cookie set
// loaded — required for endpoints that hit www.reddit.com (currently
// just PostInsights). REDDIT_COOKIES must be a JSON object mapping
// cookie name to value, e.g. {"reddit_session":"...","token_v2":"..."}.
func getTestClientWithCookies(t *testing.T) *reddit.Client {
	t.Helper()
	token := os.Getenv("REDDIT_TOKEN")
	cookiesJSON := os.Getenv("REDDIT_COOKIES")
	if token == "" || cookiesJSON == "" {
		t.Skip("REDDIT_TOKEN and REDDIT_COOKIES (JSON map) required; skipping")
	}
	var cookies map[string]string
	if err := json.Unmarshal([]byte(cookiesJSON), &cookies); err != nil {
		t.Fatalf("REDDIT_COOKIES is not valid JSON: %v", err)
	}
	return reddit.New(&reddit.Options{Token: token, Cookies: cookies})
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
	// SubredditAbout now also fetches /about/rules — guard the merge.
	if info.Rules == nil {
		t.Errorf("expected Rules to be populated by SubredditAbout, got nil")
	}
	t.Logf("  %d moderator rules, %d site rules", len(info.Rules), len(info.SiteRules))
	for _, r := range info.Rules {
		t.Logf("    [%s] %s", r.Kind, r.ShortName)
	}
}

func TestSubredditRules(t *testing.T) {
	m := getTestClient(t)
	rules, err := m.SubredditRules("sanfrancisco")
	if err != nil {
		t.Fatalf("SubredditRules() error: %v", err)
	}
	if len(rules.Rules) == 0 {
		t.Fatal("expected at least one moderator rule on r/sanfrancisco")
	}
	t.Logf("r/sanfrancisco rules: %d moderator + %d site-wide", len(rules.Rules), len(rules.SiteRules))
}

func TestPostInfoAndComments(t *testing.T) {
	m := getTestClient(t)
	// Pick a post off the user's own MyPosts feed so we can also
	// exercise the view_count code path (Reddit only returns it for
	// the authenticated user's own posts).
	mine, err := m.MyPosts(1)
	if err != nil {
		t.Fatalf("MyPosts() error: %v", err)
	}
	if len(mine.Posts) == 0 {
		t.Skip("no posts on this account; skipping post-info smoke test")
	}
	id := mine.Posts[0].ID

	info, err := m.PostInfo(id)
	if err != nil {
		t.Fatalf("PostInfo(%s) error: %v", id, err)
	}
	t.Logf("PostInfo: r/%s — %q score=%d ratio=%.2f comments=%d", info.Subreddit, info.Title, info.Score, info.UpvoteRatio, info.NumComments)
	if info.ViewCount != nil {
		t.Logf("  view_count=%d", *info.ViewCount)
	}

	post, comments, err := m.PostComments(id, &reddit.PostCommentsOptions{Limit: 5, Depth: 2})
	if err != nil {
		t.Fatalf("PostComments error: %v", err)
	}
	t.Logf("PostComments: post=%q + %d comments", post.Title, len(comments))
}

// TestPostInsights hits www.reddit.com/poststats/{id}/ and parses
// the rendered HTML. Requires REDDIT_COOKIES because the bearer
// token alone gets bounced to /login. Picks the most recent post
// off the user's own feed so the page always renders (Reddit only
// shows insights to the post author).
func TestPostInsights(t *testing.T) {
	m := getTestClientWithCookies(t)
	mine, err := m.MyPosts(1)
	if err != nil {
		t.Fatalf("MyPosts() error: %v", err)
	}
	if len(mine.Posts) == 0 {
		t.Skip("no posts on this account; skipping insights smoke test")
	}
	id := mine.Posts[0].ID

	insights, err := m.PostInsights(id)
	if err != nil {
		t.Fatalf("PostInsights(%s) error: %v", id, err)
	}
	if insights.Title == "" {
		t.Errorf("expected non-empty Title")
	}
	if insights.Subreddit == "" {
		t.Errorf("expected non-empty Subreddit")
	}
	t.Logf(
		"PostInsights[%s]: r/%s — %q",
		insights.PostID, insights.Subreddit, insights.Title,
	)
	t.Logf(
		"  views=%d (%s, change %s) • upvotes=%d (%.1f%%) • comments=%d • shares=%d • crossposts=%d • awards=%d",
		insights.TotalViews,
		insights.TotalViewsFormatted,
		insights.ViewsChangeFormatted,
		insights.Upvotes,
		insights.UpvoteRatio*100,
		insights.Comments,
		insights.Shares,
		insights.Crossposts,
		insights.Awards,
	)
	if insights.PersonalComparison != "" {
		t.Logf("  ribbon: %s", insights.PersonalComparison)
	}
	t.Logf("  hourly chart: %d hours", len(insights.HourlyViews))
	for _, c := range insights.TopComments {
		t.Logf("  top comment by u/%s (%d upvotes): %s", c.Author, c.Score, c.Body)
	}
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
