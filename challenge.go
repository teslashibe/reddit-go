package reddit

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

// Reddit fronts www.reddit.com with a lightweight JavaScript interstitial
// ("Reddit - Please wait for verification") before serving the real login
// page. A non-browser client (even with a browser TLS fingerprint) receives
// the interstitial instead of the login form, so the csrf_token cookie is
// never set and login can't proceed.
//
// The interstitial is a self-submitting form plus an inline script of the form:
//
//	document.addEventListener("DOMContentLoaded",async function(){
//	  var e=document.forms[0],
//	      n=(e.onsubmit=function(t){ /* copy location.search into hidden inputs */ },
//	         await(async e=>e+e)("c6cdff631655c0ca"));
//	  e.elements.namedItem("solution").value=n;
//	  e.requestSubmit()
//	},{once:!0});
//
//	<form hidden method="GET" action="/login/">
//	  <input type="hidden" name="solution" />
//	  <input type="hidden" name="js_challenge" value="1"/>
//	  <input type="hidden" name="token" value="7afd...."/>
//	  <input type="hidden" name="jsc_orig_r" value=""/>
//	</form>
//
// We reproduce the browser exactly: extract the inline transform lambda and its
// seed, evaluate the transform in a JS VM to compute `solution`, then re-issue
// the form's GET with solution + the hidden token fields. That request clears
// the interstitial and returns the real login page (setting csrf_token).
//
// Evaluating the lambda (rather than hard-coding today's string-doubling
// transform) keeps this resilient: Reddit rotates the seed and token on every
// request and can swap the transform body; as long as it stays a pure,
// synchronous expression we solve it without code changes.

var (
	reChallengeSolver = regexp.MustCompile(`await\((async\s+[^()]*=>[^()]+)\)\("([^"]*)"\)`)
	reChallengeForm   = regexp.MustCompile(`(?s)<form[^>]*\baction="([^"]+)"[^>]*>(.*?)</form>`)
	reHiddenInput     = regexp.MustCompile(`<input\b[^>]*\bname="([^"]+)"[^>]*\bvalue="([^"]*)"[^>]*>`)
)

// isJSChallenge reports whether body is Reddit's verification interstitial
// rather than the real page.
func isJSChallenge(body string) bool {
	return strings.Contains(body, `name="js_challenge"`) ||
		strings.Contains(body, "Please wait for verification")
}

// solveJSChallenge evaluates the interstitial's solution transform and returns
// the absolute follow-up URL (form action + computed query params) that yields
// the real page. pageURL is the URL the interstitial was served at, used to
// resolve a relative form action and to carry forward any existing query.
func solveJSChallenge(pageURL, body string) (string, error) {
	sm := reChallengeSolver.FindStringSubmatch(body)
	if sm == nil {
		return "", fmt.Errorf("challenge: solution transform not found")
	}
	lambda, seed := sm[1], sm[2]

	// Drop the leading `async` so the (pure) transform evaluates synchronously;
	// the page only marks it async so it can `await` the result.
	syncLambda := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lambda), "async"))

	vm := goja.New()
	if err := vm.Set("__seed", seed); err != nil {
		return "", fmt.Errorf("challenge: bind seed: %w", err)
	}
	v, err := vm.RunString("(" + syncLambda + ")(__seed)")
	if err != nil {
		return "", fmt.Errorf("challenge: eval solution: %w", err)
	}
	solution := v.String()

	fm := reChallengeForm.FindStringSubmatch(body)
	if fm == nil {
		return "", fmt.Errorf("challenge: self-submitting form not found")
	}
	action, inner := fm[1], fm[2]

	q := url.Values{}
	for _, m := range reHiddenInput.FindAllStringSubmatch(inner, -1) {
		q.Set(m[1], m[2])
	}
	q.Set("solution", solution)

	base, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(action)
	if err != nil {
		return "", err
	}
	resolved := base.ResolveReference(ref)
	// The page's onsubmit copies document.location.search into hidden inputs,
	// so carry forward any query params already on the served URL.
	for k, vs := range base.Query() {
		for _, val := range vs {
			q.Add(k, val)
		}
	}
	resolved.RawQuery = q.Encode()
	return resolved.String(), nil
}
