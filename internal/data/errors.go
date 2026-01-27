package data

import "errors"

var (
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrCaptchaFailed        = errors.New("captcha verification failed")
	ErrInvalidContentLength = errors.New("invalid content length")
	ErrChapterNotFound      = errors.New("chapter not found")
	ErrInvalidDisplayName   = errors.New("invalid display name")
	ErrUnsupportedFormat    = errors.New("unsupported image format")
	ErrProfileNotFound      = errors.New("profile not found")
	ErrS3NotConfigured      = errors.New("s3 not configured")
	ErrNameEmpty            = errors.New("name is empty")
	ErrNameTooLong          = errors.New("name too long")
	ErrInvalidCharacters    = errors.New("invalid characters in name")
)
