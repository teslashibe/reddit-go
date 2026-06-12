package reddit

import "errors"

var ErrRateLimited = errors.New("reddit: rate limited")

// errTwoFactor backs ErrTwoFactorRequired (defined in login.go).
var errTwoFactor = errors.New("reddit: two-factor code required (retry Login with OTP)")

// ErrRecaptchaRequired is returned by Login when no reCAPTCHA Enterprise token
// was supplied (neither LoginParams.RecaptchaToken nor RecaptchaProvider).
// Reddit's login endpoint rejects credential POSTs without one.
var ErrRecaptchaRequired = errors.New("reddit: reCAPTCHA token required (set LoginParams.RecaptchaToken or RecaptchaProvider)")
