package reddit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// MediaUpload describes a successful upload to Reddit's S3 lease bucket.
//
// S3URL is the canonical URL you pass to /api/submit as the `url` field
// when creating an image post — DO NOT substitute the i.redd.it
// equivalent. /api/submit re-fetches the URL during ingest, and the
// CDN typically 404s for a few seconds after upload, which Reddit
// surfaces as a confusing "Invalid image URL" error.
//
// AssetID + WebsocketURL are returned for callers that want to poll
// the post-creation websocket for the eventual i.redd.it URL. The
// MVP submit flow doesn't use them — `SubmitImage` returns the
// /comments/{id}/ URL directly from the submit response, which is
// what most callers actually need.
type MediaUpload struct {
	AssetID      string `json:"asset_id"`
	S3URL        string `json:"s3_url"`
	WebsocketURL string `json:"websocket_url,omitempty"`
}

// UploadMedia POSTs `data` to Reddit's media-asset lease endpoint and
// returns the canonical S3 URL the post-submission step needs.
//
// `name` is the filename Reddit will associate with the asset (used
// only for display and to derive the file extension when `mime` is
// empty). `mime` should be the canonical content-type — image/png,
// image/jpeg, image/gif, image/webp. `size` is required so we can set
// Content-Length explicitly: S3 presigned POST endpoints reject the
// chunked transfer-encoding that Go's http client falls back to when
// length is unknown.
//
// The flow is:
//
//  1. POST /api/media/asset.json with filepath+mimetype → returns the
//     S3 lease (action URL + dynamically-generated form fields).
//  2. POST the multipart payload to args.action with explicit
//     Content-Length so S3 doesn't reject for chunked encoding.
//  3. Construct the canonical S3 URL = action + "/" + s3Key, where
//     s3Key is the value of the lease field named "key".
//
// To turn the returned S3URL into a published post call
// `Client.SubmitImage`. `Client.SubmitImageFromFile` and
// `Client.SubmitImageFromURL` combine both steps.
func (c *Client) UploadMedia(name, mime string, data io.Reader, _ int64) (*MediaUpload, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("upload name is required")
	}
	mime = strings.TrimSpace(mime)
	if mime == "" {
		mime = guessMimeFromName(name)
	}
	if mime == "" {
		return nil, fmt.Errorf("could not determine mimetype for %q (pass it explicitly)", name)
	}

	leaseForm := url.Values{
		"filepath": {name},
		"mimetype": {mime},
	}
	leaseBody, err := c.oauthPost("/api/media/asset.json", leaseForm)
	if err != nil {
		return nil, fmt.Errorf("requesting media lease: %w", err)
	}

	var lease struct {
		Args struct {
			Action string `json:"action"`
			Fields []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"fields"`
		} `json:"args"`
		Asset struct {
			AssetID      string `json:"asset_id"`
			WebsocketURL string `json:"websocket_url"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(leaseBody, &lease); err != nil {
		return nil, fmt.Errorf("parsing media lease: %w", err)
	}
	if lease.Args.Action == "" || lease.Asset.AssetID == "" {
		return nil, fmt.Errorf("media lease missing action/asset_id: %s", truncate(string(leaseBody), 200))
	}
	actionURL := lease.Args.Action
	if !strings.HasPrefix(actionURL, "http") {
		// Reddit sometimes returns a protocol-relative URL like
		// "//reddit-uploaded-media.s3-accelerate.amazonaws.com".
		actionURL = "https:" + strings.TrimPrefix(actionURL, "//")
		if !strings.HasPrefix(actionURL, "https://") {
			actionURL = "https://" + lease.Args.Action
		}
	}

	// Build the multipart body in memory so we can set Content-Length
	// up front. Streaming via io.Pipe would trigger chunked encoding
	// and S3 rejects that for presigned POSTs ("Bucket POST must
	// contain a field named 'key'" — misleading, but the actual
	// problem is the chunking).
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	var s3Key string
	for _, f := range lease.Args.Fields {
		if f.Name == "key" {
			s3Key = f.Value
		}
		if err := mw.WriteField(f.Name, f.Value); err != nil {
			return nil, fmt.Errorf("writing lease field %q: %w", f.Name, err)
		}
	}
	if s3Key == "" {
		return nil, fmt.Errorf("media lease did not include an S3 key field")
	}
	fileField, err := mw.CreateFormFile("file", name)
	if err != nil {
		return nil, fmt.Errorf("creating file field: %w", err)
	}
	if _, err := io.Copy(fileField, data); err != nil {
		return nil, fmt.Errorf("copying upload bytes: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", actionURL, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("building S3 upload request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.ContentLength = int64(buf.Len())
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uploading to S3: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("S3 upload failed (status %d): %s", resp.StatusCode, truncate(string(body), 300))
	}

	return &MediaUpload{
		AssetID:      lease.Asset.AssetID,
		S3URL:        strings.TrimRight(actionURL, "/") + "/" + s3Key,
		WebsocketURL: lease.Asset.WebsocketURL,
	}, nil
}

// UploadMediaFromFile opens `localPath` and forwards to UploadMedia.
// The mimetype is inferred from the file extension; pass through
// UploadMedia directly if you need to override it.
func (c *Client) UploadMediaFromFile(localPath string) (*MediaUpload, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", localPath, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", localPath, err)
	}
	return c.UploadMedia(filepath.Base(localPath), guessMimeFromName(localPath), f, info.Size())
}

// SubmitImage publishes an image post to a subreddit. `imageURL` must
// be the S3 URL returned by UploadMedia / UploadMediaFromFile — Reddit
// re-fetches it during ingest, and i.redd.it CDN URLs aren't yet
// available at submit time.
//
// The returned Post.URL is the /comments/{id}/ permalink. The actual
// i.redd.it asset URL is delivered asynchronously over Reddit's
// post-creation websocket; SubmitImage doesn't wait for it because
// the permalink is what callers actually need to share or store.
func (c *Client) SubmitImage(subreddit, title, imageURL string) (*Post, error) {
	form := url.Values{
		"api_type":           {"json"},
		"kind":               {"image"},
		"sr":                 {subreddit},
		"title":              {title},
		"url":                {imageURL},
		// resubmit=true skips Reddit's "you've already posted this URL"
		// guard — the S3 URLs we generate are unique per upload, but
		// during dev/smoke testing it's easy to retry the same lease.
		"resubmit":           {"true"},
		"sendreplies":        {"true"},
		"validate_on_submit": {"true"},
	}
	resp, err := c.oauthPost("/api/submit", form)
	if err != nil {
		return nil, fmt.Errorf("submitting image post: %w", err)
	}
	if errs := extractSubmitErrors(resp); len(errs) > 0 {
		return nil, fmt.Errorf("reddit rejected image submission: %s", strings.Join(errs, "; "))
	}
	return parseSubmitResponse(resp)
}

// SubmitImageFromFile is a convenience that uploads `localPath` then
// publishes the result in one call. Use SubmitImage directly when the
// image is already on Reddit's S3 (e.g. a re-publish flow).
func (c *Client) SubmitImageFromFile(subreddit, title, localPath string) (*Post, error) {
	up, err := c.UploadMediaFromFile(localPath)
	if err != nil {
		return nil, err
	}
	return c.SubmitImage(subreddit, title, up.S3URL)
}

// SubmitImageFromURL fetches `imageURL` over HTTP, uploads the bytes
// to Reddit's S3 lease bucket, and publishes the post. Use when the
// image is hosted by an in-product upload endpoint rather than a
// local file — typical for chat surfaces where the user attaches an
// image and the backend exposes it via a signed URL.
//
// The fetch is plain HTTP GET with no auth headers. If you need to
// attach auth, fetch yourself and call UploadMedia + SubmitImage
// directly.
func (c *Client) SubmitImageFromURL(subreddit, title, imageURL string) (*Post, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building image fetch request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching image from %s: %w", imageURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("image fetch failed (status %d): %s", resp.StatusCode, truncate(string(body), 200))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading image bytes: %w", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("image at %s is empty", imageURL)
	}
	mime := strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0])
	if mime == "" {
		mime = http.DetectContentType(body)
	}
	if !strings.HasPrefix(mime, "image/") {
		return nil, fmt.Errorf("URL %s did not return an image (got Content-Type %q)", imageURL, mime)
	}

	name := imageNameFromURL(imageURL, mime)
	up, err := c.UploadMedia(name, mime, bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	return c.SubmitImage(subreddit, title, up.S3URL)
}

// extractSubmitErrors pulls the (errcode, message, fieldname) tuples
// Reddit returns inside an otherwise-200 /api/submit response. Reddit
// signals user-facing failures (NO_TEXT, SUBREDDIT_NOEXIST, spam
// filter trips) here rather than via HTTP status, so a successful
// HTTP 200 still needs to be checked.
func extractSubmitErrors(resp []byte) []string {
	var result struct {
		JSON struct {
			Errors [][]string `json:"errors"`
		} `json:"json"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil
	}
	if len(result.JSON.Errors) == 0 {
		return nil
	}
	out := make([]string, 0, len(result.JSON.Errors))
	for _, e := range result.JSON.Errors {
		switch len(e) {
		case 0:
			continue
		case 1:
			out = append(out, e[0])
		default:
			out = append(out, fmt.Sprintf("%s: %s", e[0], e[1]))
		}
	}
	return out
}

// guessMimeFromName returns the canonical Reddit-acceptable content-type
// for a filename based on its extension, or "" for unrecognized
// extensions. Only image types are listed because the lease endpoint
// rejects unknown mimetypes anyway.
func guessMimeFromName(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	}
	return ""
}

// imageNameFromURL produces a sensible filename for a fetched image:
// take the URL's basename, strip any query string, and ensure the
// extension matches the actual mimetype. Reddit's lease endpoint uses
// the extension to validate the upload, so a mismatched name (e.g.
// "image" with no extension when mimetype is image/png) silently
// produces a 400.
func imageNameFromURL(rawURL, mime string) string {
	name := "image"
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		base := path.Base(u.Path)
		if base != "" && base != "." && base != "/" {
			name = base
		}
	}
	wantExt := ""
	switch mime {
	case "image/png":
		wantExt = ".png"
	case "image/jpeg":
		wantExt = ".jpg"
	case "image/gif":
		wantExt = ".gif"
	case "image/webp":
		wantExt = ".webp"
	}
	if wantExt == "" {
		return name
	}
	cur := strings.ToLower(filepath.Ext(name))
	if cur == wantExt || (wantExt == ".jpg" && cur == ".jpeg") {
		return name
	}
	return strings.TrimSuffix(name, cur) + wantExt
}
