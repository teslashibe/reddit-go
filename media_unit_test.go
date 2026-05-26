package reddit

import "testing"

func TestGuessMimeFromName(t *testing.T) {
	cases := map[string]string{
		"foo.png":   "image/png",
		"FOO.PNG":   "image/png",
		"a.b.JPEG":  "image/jpeg",
		"thing.jpg": "image/jpeg",
		"x.gif":     "image/gif",
		"y.webp":    "image/webp",
		"no-ext":    "",
		"x.bmp":     "",
	}
	for in, want := range cases {
		if got := guessMimeFromName(in); got != want {
			t.Errorf("guessMimeFromName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestImageNameFromURL(t *testing.T) {
	cases := []struct {
		url  string
		mime string
		want string
	}{
		// Common case: filename has the right extension already.
		{"https://example.com/uploads/cat.png", "image/png", "cat.png"},
		// Filename has wrong extension — replace it to match mime.
		{"https://example.com/uploads/cat.jpg", "image/png", "cat.png"},
		// jpeg/jpg are interchangeable.
		{"https://example.com/photo.jpeg", "image/jpeg", "photo.jpeg"},
		// Query string should be stripped.
		{"https://example.com/api/uploads/abc?sig=xyz", "image/png", "abc.png"},
		// No extension, no path → fallback name.
		{"https://example.com/", "image/jpeg", "image.jpg"},
		// Unknown mime → leave name as-is.
		{"https://example.com/foo.bar", "application/octet-stream", "foo.bar"},
		// Path basename that's a dot/slash sentinel falls back.
		{"https://example.com//", "image/png", "image.png"},
	}
	for _, c := range cases {
		if got := imageNameFromURL(c.url, c.mime); got != c.want {
			t.Errorf("imageNameFromURL(%q, %q) = %q, want %q", c.url, c.mime, got, c.want)
		}
	}
}

func TestExtractSubmitErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "no errors",
			body: `{"json":{"errors":[],"data":{"id":"abc"}}}`,
			want: nil,
		},
		{
			name: "single triple",
			body: `{"json":{"errors":[["NO_TEXT","we need a body","title"]]}}`,
			want: []string{"NO_TEXT: we need a body"},
		},
		{
			name: "multiple",
			body: `{"json":{"errors":[["SUBREDDIT_NOEXIST","that subreddit doesn't exist","sr"],["BAD_IMAGE","invalid image","url"]]}}`,
			want: []string{
				"SUBREDDIT_NOEXIST: that subreddit doesn't exist",
				"BAD_IMAGE: invalid image",
			},
		},
		{
			name: "single value",
			body: `{"json":{"errors":[["LONELY"]]}}`,
			want: []string{"LONELY"},
		},
		{
			name: "malformed json returns nil",
			body: `not json`,
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractSubmitErrors([]byte(c.body))
			if len(got) != len(c.want) {
				t.Fatalf("len = %d, want %d (got %v)", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}
