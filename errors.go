package reddit

import "errors"

var ErrRateLimited = errors.New("reddit: rate limited")
