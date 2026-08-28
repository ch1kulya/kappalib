package data

import "errors"

var (
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrCaptchaFailed         = errors.New("captcha verification failed")
	ErrInvalidContentLength  = errors.New("invalid content length")
	ErrChapterNotFound       = errors.New("chapter not found")
	ErrInvalidDisplayName    = errors.New("invalid display name")
	ErrUnsupportedFormat     = errors.New("unsupported image format")
	ErrProfileNotFound       = errors.New("profile not found")
	ErrS3NotConfigured       = errors.New("s3 not configured")
	ErrNameEmpty             = errors.New("name is empty")
	ErrNameTooLong           = errors.New("name too long")
	ErrInvalidCharacters     = errors.New("invalid characters in name")
	ErrInvalidVoteValue      = errors.New("invalid vote value")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrNotCommentAuthor      = errors.New("not comment author")
	ErrCannotDeleteComment   = errors.New("cannot delete comment with current status")
	ErrCommentAnswerNotFound = errors.New("comment answer not found")
	ErrNotAnswerAuthor       = errors.New("not answer author")
	ErrCannotDeleteAnswer    = errors.New("cannot delete answer with current status")
	ErrInvalidAnswerLength   = errors.New("invalid answer content length")
	ErrCommentNotApproved    = errors.New("comment must be approved to answer")
	ErrCannotEditComment     = errors.New("cannot edit comment with current status")
	ErrCannotEditAnswer      = errors.New("cannot edit answer with current status")
	ErrBookmarkNotFound      = errors.New("bookmark not found")
)
