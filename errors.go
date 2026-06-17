package reddit

import "errors"

var ErrRateLimited = errors.New("reddit: rate limited")

// ErrUnauthorized is returned (wrapped) when Reddit rejects the bearer token:
// an OAuth/web 401/403, or a Matrix response carrying M_UNKNOWN_TOKEN. The
// session bearer (token_v2) is shared across oauth.reddit.com and
// matrix.redditspace.com, but Reddit's Matrix gateway can drop a token's chat
// session before OAuth stops honoring it — so chat starts returning
// M_UNKNOWN_TOKEN while REST still works. Hosts can errors.Is(err, ErrUnauthorized)
// to self-heal by re-minting the session (a fresh login) and retrying.
var ErrUnauthorized = errors.New("reddit: unauthorized (token expired or invalid)")

// errTwoFactor backs ErrTwoFactorRequired (defined in login.go).
var errTwoFactor = errors.New("reddit: two-factor code required (retry Login with OTP)")

// ErrRecaptchaRequired is returned by Login when no reCAPTCHA Enterprise token
// was supplied (neither LoginParams.RecaptchaToken nor RecaptchaProvider).
// Reddit's login endpoint rejects credential POSTs without one.
var ErrRecaptchaRequired = errors.New("reddit: reCAPTCHA token required (set LoginParams.RecaptchaToken or RecaptchaProvider)")
