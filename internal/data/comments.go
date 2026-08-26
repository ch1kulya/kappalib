package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	nethtml "golang.org/x/net/html"

	"github.com/ch1kulya/logger"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

//go:embed sql/comments_create.sql
var queryCommentsCreate string

//go:embed sql/comments_edit.sql
var queryCommentsEdit string

//go:embed sql/comments_get_by_id.sql
var queryCommentsGetByID string

//go:embed sql/comments_update_status.sql
var queryCommentsUpdateStatus string

//go:embed sql/comments_set_telegram_message_id.sql
var queryCommentsSetTelegramMessageID string

//go:embed sql/comment_votes_upsert.sql
var queryCommentVotesUpsert string

//go:embed sql/comment_votes_delete.sql
var queryCommentVotesDelete string

//go:embed sql/comment_votes_score.sql
var queryCommentVotesScore string

//go:embed sql/comment_answers_create.sql
var queryCommentAnswersCreate string

//go:embed sql/comment_answers_edit.sql
var queryCommentAnswersEdit string

//go:embed sql/comment_answers_update_status.sql
var queryCommentAnswersUpdateStatus string

//go:embed sql/comment_answers_delete_by_comment_user.sql
var queryCommentAnswersDeleteByCommentUser string

//go:embed sql/comment_answers_set_telegram_message_id.sql
var queryCommentAnswersSetTelegramMessageID string

//go:embed sql/comments_telegram_info.sql
var queryCommentsTelegramInfo string

//go:embed sql/comment_answers_telegram_info.sql
var queryCommentAnswersTelegramInfo string

//go:embed sql/user_comment_threads.sql
var queryUserCommentThreads string

//go:embed sql/comment_stats_base.sql
var queryCommentStatsBase string

//go:embed sql/comment_stats_daily.sql
var queryCommentStatsDaily string

var (
	commentsTurnstileSecret    = os.Getenv("TURNSTILE_COMMENTS_SECRET")
	commentsSmartCaptchaSecret = os.Getenv("SMARTCAPTCHA_SECRET")
	telegramBotToken           = os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID             = os.Getenv("TELEGRAM_CHAT_ID")
	telegramWebhookSecret      = os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	markdownPolicy             *bluemonday.Policy
	spoilerRegex               = regexp.MustCompile(`(?s)\|\|(.+?)\|\|`)
	telegramClient             = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			IdleConnTimeout:       30 * time.Second,
		},
	}
)

var userCommentLimiter = struct {
	sync.Mutex
	lastComment map[string]time.Time
}{
	lastComment: make(map[string]time.Time),
}

const commentCooldown = 30 * time.Second

func checkCommentRateLimit(userID string) bool {
	userCommentLimiter.Lock()
	defer userCommentLimiter.Unlock()

	if last, exists := userCommentLimiter.lastComment[userID]; exists {
		if time.Since(last) < commentCooldown {
			return false
		}
	}
	return true
}

func recordCommentTime(userID string) {
	userCommentLimiter.Lock()
	defer userCommentLimiter.Unlock()
	userCommentLimiter.lastComment[userID] = time.Now()
}

func init() {
	markdownPolicy = bluemonday.NewPolicy()
	markdownPolicy.AllowStandardURLs()
	markdownPolicy.AllowRelativeURLs(false)
	markdownPolicy.RequireNoFollowOnLinks(true)
	markdownPolicy.RequireNoReferrerOnLinks(true)
	markdownPolicy.AllowElements("p", "br", "strong", "b", "em", "i", "code", "pre", "blockquote")
	markdownPolicy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	markdownPolicy.AllowElements("ul", "ol", "li")
	markdownPolicy.AllowAttrs("class").Matching(regexp.MustCompile(`^spoiler$`)).OnElements("span")
	markdownPolicy.AllowAttrs("href").OnElements("a")
	markdownPolicy.AllowURLSchemes("http", "https")
	markdownPolicy.AllowImages()
	markdownPolicy.AllowAttrs("src", "alt", "title", "loading").OnElements("img")
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			userCommentLimiter.Lock()
			now := time.Now()
			for userID, last := range userCommentLimiter.lastComment {
				if now.Sub(last) > 5*time.Minute {
					delete(userCommentLimiter.lastComment, userID)
				}
			}
			userCommentLimiter.Unlock()

			imageUploadLimiter.Lock()
			for userID, last := range imageUploadLimiter.lastUpload {
				if now.Sub(last) > 5*time.Minute {
					delete(imageUploadLimiter.lastUpload, userID)
				}
			}
			imageUploadLimiter.Unlock()
		}
	}()
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	useSSL := os.Getenv("S3_USE_SSL") != "false"

	if endpoint != "" && accessKey != "" && secretKey != "" {
		endpoint = strings.TrimPrefix(endpoint, "https://")
		endpoint = strings.TrimPrefix(endpoint, "http://")

		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		})
		if err != nil {
			logger.Error("Failed to initialize MinIO client: %v", err)
		} else {
			minioClient = client
			logger.Info("MinIO client initialized for endpoint: %s", endpoint)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_, err := client.BucketExists(ctx, s3Bucket)
				if err != nil {
					logger.Warn("S3 warmup failed: %v", err)
				} else {
					logger.Info("S3 connection warmed up")
				}
			}()
		}
	}
}

func verifyCommentsTurnstile(token string) bool {
	if commentsTurnstileSecret == "" {
		logger.Warn("TURNSTILE_COMMENTS_SECRET not set")
		return false
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		map[string][]string{
			"secret":   {commentsTurnstileSecret},
			"response": {token},
		})
	if err != nil {
		logger.Error("Comments turnstile verification failed: %v", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}
	return result.Success
}

func verifyCommentsSmartCaptcha(token, ip string) bool {
	if commentsSmartCaptchaSecret == "" {
		logger.Warn("SMARTCAPTCHA_SECRET not set")
		return false
	}

	client := &http.Client{Timeout: 3 * time.Second}
	params := url.Values{
		"secret": {commentsSmartCaptchaSecret},
		"token":  {token},
	}
	if ip != "" {
		params.Set("ip", ip)
	}

	resp, err := client.PostForm("https://smartcaptcha.yandexcloud.net/validate", params)
	if err != nil {
		logger.Error("Comments smartcaptcha verification request failed: %v", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Smartcaptcha server returned status %d", resp.StatusCode)
		return false
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("Failed to decode smartcaptcha response: %v", err)
		return false
	}
	return result.Status == "ok"
}

func verifyCommentsCaptcha(turnstileToken, smartCaptchaToken, ip string) bool {
	if smartCaptchaToken != "" {
		return verifyCommentsSmartCaptcha(smartCaptchaToken, ip)
	}
	if turnstileToken != "" {
		return verifyCommentsTurnstile(turnstileToken)
	}
	return false
}

func renderMarkdown(content string) string {
	unsafe := blackfriday.Run([]byte(content),
		blackfriday.WithExtensions(blackfriday.CommonExtensions&^blackfriday.Tables&^blackfriday.FencedCode),
	)
	safe := markdownPolicy.SanitizeBytes(unsafe)
	result := spoilerRegex.ReplaceAllString(string(safe), `<span class="spoiler">$1</span>`)
	result = strings.ReplaceAll(result, "<img src=", `<img loading="lazy" src=`)
	return strings.TrimSpace(result)
}

func validateSubmission(userID, content string, maxLen int, isAnswer bool, turnstileToken, smartCaptchaToken, ip string) error {
	contentLen := utf8.RuneCountInString(content)
	if contentLen == 0 || contentLen > maxLen {
		if isAnswer {
			return ErrInvalidAnswerLength
		}
		return ErrInvalidContentLength
	}

	if !checkCommentRateLimit(userID) {
		return ErrRateLimitExceeded
	}

	if !verifyCommentsCaptcha(turnstileToken, smartCaptchaToken, ip) {
		return ErrCaptchaFailed
	}

	return nil
}

func CreateComment(ctx context.Context, userID string, input models.CreateCommentInput) (*models.Comment, error) {
	if err := validateSubmission(userID, input.Content, 3000, false, input.TurnstileToken, input.SmartCaptchaToken, input.IP); err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !chapterExists(dbCtx, input.ChapterID) {
		return nil, ErrChapterNotFound
	}

	contentHTML := renderMarkdown(input.Content)

	var comment models.Comment
	err := database.DB.QueryRow(dbCtx, queryCommentsCreate,
		input.ChapterID, userID, contentHTML,
	).Scan(&comment.ID, &comment.ChapterID, &comment.UserID, &comment.ContentHTML, &comment.Status, &comment.EditedAt, &comment.CreatedAt)

	if err != nil {
		logger.Error("Failed to create comment: %v", err)
		return nil, err
	}

	var user models.ProfilePublic
	var avatarUpdatedAt time.Time
	if err := database.DB.QueryRow(dbCtx, `SELECT display_name, avatar_seed, has_custom_avatar, avatar_updated_at FROM users WHERE id = $1`, userID).Scan(&user.DisplayName, &user.AvatarSeed, &user.HasCustomAvatar, &avatarUpdatedAt); err != nil {
		logger.Warn("Failed to fetch user data for comment: %v", err)
	}
	comment.UserDisplayName = user.DisplayName
	comment.UserAvatarSeed = user.AvatarSeed
	comment.UserHasCustomAvatar = user.HasCustomAvatar
	comment.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()

	go sendCommentToTelegram(context.Background(), &comment)

	recordCommentTime(userID)

	logger.Info("Comment created: %s by user %s", comment.ID, userID)
	return &comment, nil
}

func EditComment(ctx context.Context, commentID, userID string, input models.EditCommentInput) (*models.Comment, error) {
	if err := validateSubmission(userID, input.Content, 3000, false, input.TurnstileToken, input.SmartCaptchaToken, input.IP); err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ownerID, status, oldContentHTML string
	err := database.DB.QueryRow(dbCtx, `SELECT user_id, status, content_html FROM comments WHERE id = $1`, commentID).Scan(&ownerID, &status, &oldContentHTML)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	if ownerID != userID {
		return nil, ErrNotCommentAuthor
	}

	if status != "approved" && status != "rejected" {
		return nil, ErrCannotEditComment
	}

	contentHTML := renderMarkdown(input.Content)

	var comment models.Comment
	err = database.DB.QueryRow(dbCtx, queryCommentsEdit,
		contentHTML, commentID, userID,
	).Scan(&comment.ID, &comment.ChapterID, &comment.UserID, &comment.ContentHTML, &comment.Status, &comment.EditedAt, &comment.CreatedAt)

	if err != nil {
		logger.Error("Failed to edit comment: %v", err)
		return nil, err
	}

	var user models.ProfilePublic
	var avatarUpdatedAt time.Time
	if err := database.DB.QueryRow(dbCtx, `SELECT display_name, avatar_seed, has_custom_avatar, avatar_updated_at FROM users WHERE id = $1`, userID).Scan(&user.DisplayName, &user.AvatarSeed, &user.HasCustomAvatar, &avatarUpdatedAt); err != nil {
		logger.Warn("Failed to fetch user data for edited comment: %v", err)
	}
	comment.UserDisplayName = user.DisplayName
	comment.UserAvatarSeed = user.AvatarSeed
	comment.UserHasCustomAvatar = user.HasCustomAvatar
	comment.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()

	go sendEditedCommentToTelegram(context.Background(), &comment, oldContentHTML)

	recordCommentTime(userID)

	logger.Info("Comment %s edited by user %s", comment.ID, userID)
	return &comment, nil
}

func CalculateCommentsPagination(page, pageSize, totalCount int, isDeepLink bool, targetPage int) (limit, offset, resultPage, totalPages int) {
	if totalCount > 0 && pageSize > 0 {
		totalPages = (totalCount + pageSize - 1) / pageSize
	}
	if isDeepLink {
		return targetPage * pageSize, 0, targetPage, totalPages
	}
	return pageSize, (page - 1) * pageSize, page, totalPages
}

func GetVisibleComments(ctx context.Context, chapterID, userID string, page int, targetID ...string) (*models.CommentsPage, error) {
	pageSize := 12

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var target string
	if len(targetID) > 0 {
		target = targetID[0]
	}

	targetPage := page
	isDeepLink := false

	if target != "" {
		targetCommentID := target
		if strings.HasPrefix(target, "can_") {
			var parentCommentID string
			err := database.DB.QueryRow(dbCtx, `
				SELECT comment_id
				FROM comment_answers
				WHERE id = $1
				  AND status != 'deleted'
				  AND (status = 'approved' OR (status IN ('pending', 'rejected') AND user_id = $2))
			`, target, userID).Scan(&parentCommentID)
			if err == nil && parentCommentID != "" {
				targetCommentID = parentCommentID
			} else {
				targetCommentID = ""
			}
		}

		if targetCommentID != "" {
			var targetCreatedAt time.Time
			err := database.DB.QueryRow(dbCtx, `
				SELECT created_at
				FROM comments
				WHERE id = $1
				  AND chapter_id = $2
				  AND status != 'deleted'
				  AND (status = 'approved' OR (status IN ('pending', 'rejected') AND user_id = $3))
			`, targetCommentID, chapterID, userID).Scan(&targetCreatedAt)
			if err == nil {
				var pos int
				err = database.DB.QueryRow(dbCtx, `
					SELECT COUNT(*)
					FROM comments c
					WHERE c.chapter_id = $1
					  AND c.status != 'deleted'
					  AND (
					    c.status = 'approved'
					    OR (c.status IN ('pending', 'rejected') AND c.user_id = $2)
					  )
					  AND (c.created_at > $3 OR (c.created_at = $3 AND c.id < $4))
				`, chapterID, userID, targetCreatedAt, targetCommentID).Scan(&pos)
				if err == nil {
					targetPage = (pos / pageSize) + 1
					isDeepLink = true
				}
			}
		}
	}

	var topLevelCount, answersCount int
	err := database.DB.QueryRow(dbCtx, `
        SELECT
            COUNT(DISTINCT c.id) AS top_level_count,
            COUNT(DISTINCT ca.id) AS answers_count
        FROM comments c
        LEFT JOIN comment_answers ca ON ca.comment_id = c.id
            AND ca.status != 'deleted'
            AND (
                ca.status = 'approved'
                OR (ca.status IN ('pending', 'rejected') AND ca.user_id = $2)
            )
        WHERE c.chapter_id = $1
          AND c.status != 'deleted'
          AND (
            c.status = 'approved'
            OR (c.status IN ('pending', 'rejected') AND c.user_id = $2)
          )
    `, chapterID, userID).Scan(&topLevelCount, &answersCount)
	if err != nil {
		logger.Error("Failed to count visible comments: %v", err)
		return nil, err
	}

	if topLevelCount == 0 {
		return &models.CommentsPage{
			Comments:   []models.Comment{},
			Page:       1,
			PageSize:   pageSize,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	totalCount := topLevelCount + answersCount

	limit, offset, resultPage, totalPages := CalculateCommentsPagination(page, pageSize, topLevelCount, isDeepLink, targetPage)

	rows, err := database.DB.Query(dbCtx, `
        SELECT
            c.id, c.chapter_id, c.user_id, c.content_html, c.status, c.edited_at, c.created_at,
            u.display_name, u.avatar_seed, u.has_custom_avatar, u.avatar_updated_at,
            COALESCE((SELECT SUM(value) FROM comment_votes WHERE comment_id = c.id), 0)::int,
            COALESCE((SELECT value FROM comment_votes WHERE comment_id = c.id AND user_id = $2), 0)::int
        FROM comments c
        JOIN users u ON c.user_id = u.id
        WHERE c.chapter_id = $1
          AND c.status != 'deleted'
          AND (
            c.status = 'approved'
            OR (c.status IN ('pending', 'rejected') AND c.user_id = $2)
          )
        ORDER BY c.created_at DESC, c.id ASC
        LIMIT $3 OFFSET $4
    `, chapterID, userID, limit, offset)
	if err != nil {
		logger.Error("Failed to get visible comments: %v", err)
		return nil, err
	}
	defer rows.Close()

	comments := make([]models.Comment, 0)
	for rows.Next() {
		var c models.Comment
		var avatarUpdatedAt time.Time
		if err := rows.Scan(
			&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status, &c.EditedAt,
			&c.CreatedAt, &c.UserDisplayName, &c.UserAvatarSeed,
			&c.UserHasCustomAvatar, &avatarUpdatedAt,
			&c.Score, &c.UserVote,
		); err != nil {
			logger.Warn("Comment row scan error: %v", err)
			continue
		}
		c.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()
		comments = append(comments, c)
	}

	if len(comments) > 0 {
		commentIDs := make([]string, len(comments))
		for i, c := range comments {
			commentIDs[i] = c.ID
		}

		answerRows, err := database.DB.Query(dbCtx, `
			SELECT
				ca.id, ca.comment_id, ca.user_id, ca.content_html, ca.status, ca.edited_at, ca.created_at,
				u.display_name, u.avatar_seed, u.has_custom_avatar, u.avatar_updated_at
			FROM comment_answers ca
			JOIN users u ON ca.user_id = u.id
			WHERE ca.comment_id = ANY($1)
			  AND ca.status != 'deleted'
			  AND (
			    ca.status = 'approved'
			    OR (ca.status IN ('pending', 'rejected') AND ca.user_id = $2)
			  )
			ORDER BY ca.comment_id, ca.created_at ASC
		`, commentIDs, userID)
		if err != nil {
			logger.Warn("Failed to batch-load comment answers: %v", err)
		} else {
			defer answerRows.Close()

			answersMap := make(map[string][]models.CommentAnswer)
			for answerRows.Next() {
				var a models.CommentAnswer
				var avatarUpdatedAt time.Time
				if err := answerRows.Scan(
					&a.ID, &a.CommentID, &a.UserID, &a.ContentHTML, &a.Status, &a.EditedAt,
					&a.CreatedAt, &a.UserDisplayName, &a.UserAvatarSeed,
					&a.UserHasCustomAvatar, &avatarUpdatedAt,
				); err != nil {
					logger.Warn("Answer row scan error: %v", err)
					continue
				}
				a.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()
				answersMap[a.CommentID] = append(answersMap[a.CommentID], a)
			}

			for i := range comments {
				if ans, ok := answersMap[comments[i].ID]; ok {
					comments[i].Answers = ans
				}
			}
		}
	}

	return &models.CommentsPage{
		Comments:   comments,
		Page:       resultPage,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func GetUserComments(ctx context.Context, userID string, page int) (*models.UserCommentsPage, error) {
	pageSize := 12
	offset := (page - 1) * pageSize

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type idWithDate struct {
		ID        string
		CreatedAt time.Time
	}

	rows, err := database.DB.Query(dbCtx, queryUserCommentThreads, userID, pageSize, offset)
	if err != nil {
		logger.Error("Failed to get user comment threads: %v", err)
		return nil, err
	}
	defer rows.Close()

	var items []idWithDate
	for rows.Next() {
		var item idWithDate
		if err := rows.Scan(&item.ID, &item.CreatedAt); err != nil {
			logger.Warn("Thread row scan error: %v", err)
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return &models.UserCommentsPage{
			Comments:   []models.Comment{},
			Page:       page,
			PageSize:   pageSize,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	commentIDs := make([]string, len(items))
	for i, item := range items {
		commentIDs[i] = item.ID
	}

	commentsByID := make(map[string]*models.Comment, len(commentIDs))

	commentRows, err := database.DB.Query(dbCtx, `
		SELECT
			c.id, c.chapter_id, c.user_id, c.content_html, c.status, c.edited_at, c.created_at,
			u.display_name, u.avatar_seed, u.has_custom_avatar, u.avatar_updated_at,
			COALESCE((SELECT SUM(value) FROM comment_votes WHERE comment_id = c.id), 0)::int,
			COALESCE((SELECT value FROM comment_votes WHERE comment_id = c.id AND user_id = $2), 0)::int,
			ch.chapter_num, ch.novel_id,
			n.title
		FROM comments c
		JOIN users u ON c.user_id = u.id
		JOIN chapters ch ON c.chapter_id = ch.id
		JOIN novels n ON ch.novel_id = n.id
		WHERE c.id = ANY($1)
	`, commentIDs, userID)
	if err != nil {
		logger.Error("Failed to get user thread comments: %v", err)
		return nil, err
	}
	defer commentRows.Close()

	for commentRows.Next() {
		var c models.Comment
		var avatarUpdatedAt time.Time
		if err := commentRows.Scan(
			&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status, &c.EditedAt,
			&c.CreatedAt, &c.UserDisplayName, &c.UserAvatarSeed,
			&c.UserHasCustomAvatar, &avatarUpdatedAt,
			&c.Score, &c.UserVote,
			&c.ChapterNum, &c.NovelID, &c.NovelTitle,
		); err != nil {
			logger.Warn("Comment row scan error: %v", err)
			continue
		}
		c.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()
		commentsByID[c.ID] = &c
	}

	answerRows, err := database.DB.Query(dbCtx, `
		SELECT
			ca.id, ca.comment_id, ca.user_id, ca.content_html, ca.status, ca.edited_at, ca.created_at,
			u.display_name, u.avatar_seed, u.has_custom_avatar, u.avatar_updated_at
		FROM comment_answers ca
		JOIN users u ON ca.user_id = u.id
		WHERE ca.comment_id = ANY($1)
		  AND ca.status != 'deleted'
		  AND (
		    ca.status = 'approved'
		    OR (ca.status IN ('pending', 'rejected') AND ca.user_id = $2)
		  )
		ORDER BY ca.comment_id, ca.created_at ASC
	`, commentIDs, userID)
	if err != nil {
		logger.Warn("Failed to batch-load answers for user threads: %v", err)
	} else {
		var lastSeen sql.NullTime
		_ = database.DB.QueryRow(dbCtx,
			`SELECT notifications_last_seen FROM users WHERE id = $1`, userID,
		).Scan(&lastSeen)

		notifyThreshold := time.Now().AddDate(-1, 0, 0)
		if lastSeen.Valid {
			notifyThreshold = lastSeen.Time
		}

		defer answerRows.Close()
		for answerRows.Next() {
			var a models.CommentAnswer
			var avatarUpdatedAt time.Time
			if err := answerRows.Scan(
				&a.ID, &a.CommentID, &a.UserID, &a.ContentHTML, &a.Status, &a.EditedAt,
				&a.CreatedAt, &a.UserDisplayName, &a.UserAvatarSeed,
				&a.UserHasCustomAvatar, &avatarUpdatedAt,
			); err != nil {
				logger.Warn("Answer row scan error: %v", err)
				continue
			}
			a.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()
			a.IsNew = a.UserID != userID && a.CreatedAt.After(notifyThreshold)

			if c, ok := commentsByID[a.CommentID]; ok {
				c.Answers = append(c.Answers, a)
			}
		}
	}

	comments := make([]models.Comment, 0, len(items))
	for _, item := range items {
		if c, ok := commentsByID[item.ID]; ok {
			comments = append(comments, *c)
		}
	}

	var totalCount int
	err = database.DB.QueryRow(dbCtx, `
		SELECT COUNT(*) FROM (
			SELECT id FROM comments WHERE user_id = $1 AND status != 'deleted'
			UNION
			SELECT comment_id FROM comment_answers WHERE user_id = $1 AND status != 'deleted'
		) AS combined
	`, userID).Scan(&totalCount)
	if err != nil {
		logger.Error("Failed to count user threads: %v", err)
		totalCount = len(comments)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	return &models.UserCommentsPage{
		Comments:   comments,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func VoteComment(ctx context.Context, commentID, userID string, value int) (int, error) {
	if value != -1 && value != 0 && value != 1 {
		return 0, ErrInvalidVoteValue
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exists bool
	err := database.DB.QueryRow(dbCtx,
		`SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1 AND (status = 'approved' OR (status = 'pending' AND user_id = $2)))`,
		commentID, userID,
	).Scan(&exists)
	if err != nil || !exists {
		return 0, ErrCommentNotFound
	}

	if value == 0 {
		_, err = database.DB.Exec(dbCtx, queryCommentVotesDelete, commentID, userID)
	} else {
		_, err = database.DB.Exec(dbCtx, queryCommentVotesUpsert, commentID, userID, value)
	}
	if err != nil {
		logger.Error("Failed to vote on comment: %v", err)
		return 0, err
	}

	var score int
	err = database.DB.QueryRow(dbCtx, queryCommentVotesScore, commentID).Scan(&score)
	if err != nil {
		return 0, err
	}

	return score, nil
}

func UpdateCommentStatus(ctx context.Context, commentID, status string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var id string
	err := database.DB.QueryRow(dbCtx, queryCommentsUpdateStatus, status, commentID).Scan(&id)
	if err != nil {
		logger.Error("Failed to update comment status: %v", err)
		return err
	}

	logger.Info("Comment %s status updated to %s", commentID, status)
	return nil
}

func DeleteComment(ctx context.Context, commentID, userID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ownerID string
	var status string
	err := database.DB.QueryRow(dbCtx,
		`SELECT user_id, status FROM comments WHERE id = $1`,
		commentID,
	).Scan(&ownerID, &status)
	if err != nil {
		return ErrCommentNotFound
	}

	if ownerID != userID {
		return ErrNotCommentAuthor
	}

	if status != "approved" {
		return ErrCannotDeleteComment
	}

	if err := UpdateCommentStatus(ctx, commentID, "deleted"); err != nil {
		return err
	}

	if _, err := database.DB.Exec(dbCtx, queryCommentAnswersDeleteByCommentUser, commentID, userID); err != nil {
		logger.Error("Failed to delete user answers for comment %s: %v", commentID, err)
		return err
	}

	logger.Info("Comment %s deleted by user %s", commentID, userID)
	return nil
}

func GetCommentByID(ctx context.Context, commentID string) (*models.Comment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var c models.Comment
	err := database.DB.QueryRow(dbCtx, queryCommentsGetByID, commentID).Scan(
		&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status, &c.EditedAt, &c.TelegramMessageID, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func CreateCommentAnswer(ctx context.Context, userID string, input models.CreateCommentAnswerInput) (*models.CommentAnswer, error) {
	if err := validateSubmission(userID, input.Content, 500, true, input.TurnstileToken, input.SmartCaptchaToken, input.IP); err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var commentStatus string
	err := database.DB.QueryRow(dbCtx,
		`SELECT status FROM comments WHERE id = $1`,
		input.CommentID,
	).Scan(&commentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	if commentStatus != "approved" {
		return nil, ErrCommentNotApproved
	}

	contentHTML := renderMarkdown(input.Content)

	var answer models.CommentAnswer
	err = database.DB.QueryRow(dbCtx, queryCommentAnswersCreate,
		input.CommentID, userID, contentHTML,
	).Scan(&answer.ID, &answer.CommentID, &answer.UserID, &answer.ContentHTML, &answer.Status, &answer.EditedAt, &answer.CreatedAt)

	if err != nil {
		logger.Error("Failed to create comment answer: %v", err)
		return nil, err
	}

	var user models.ProfilePublic
	var avatarUpdatedAt time.Time
	if err := database.DB.QueryRow(dbCtx, `SELECT display_name, avatar_seed, has_custom_avatar, avatar_updated_at FROM users WHERE id = $1`, userID).Scan(&user.DisplayName, &user.AvatarSeed, &user.HasCustomAvatar, &avatarUpdatedAt); err != nil {
		logger.Warn("Failed to fetch user data for answer: %v", err)
	}
	answer.UserDisplayName = user.DisplayName
	answer.UserAvatarSeed = user.AvatarSeed
	answer.UserHasCustomAvatar = user.HasCustomAvatar
	answer.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()

	go sendAnswerToTelegram(context.Background(), &answer)

	recordCommentTime(userID)

	logger.Info("Comment answer created: %s by user %s on comment %s", answer.ID, userID, input.CommentID)
	return &answer, nil
}

func EditCommentAnswer(ctx context.Context, answerID, userID string, input models.EditCommentAnswerInput) (*models.CommentAnswer, error) {
	if err := validateSubmission(userID, input.Content, 500, true, input.TurnstileToken, input.SmartCaptchaToken, input.IP); err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ownerID, status, oldContentHTML string
	err := database.DB.QueryRow(dbCtx, `SELECT user_id, status, content_html FROM comment_answers WHERE id = $1`, answerID).Scan(&ownerID, &status, &oldContentHTML)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentAnswerNotFound
		}
		return nil, err
	}

	if ownerID != userID {
		return nil, ErrNotAnswerAuthor
	}

	if status != "approved" && status != "rejected" {
		return nil, ErrCannotEditAnswer
	}

	contentHTML := renderMarkdown(input.Content)

	var answer models.CommentAnswer
	err = database.DB.QueryRow(dbCtx, queryCommentAnswersEdit,
		contentHTML, answerID, userID,
	).Scan(&answer.ID, &answer.CommentID, &answer.UserID, &answer.ContentHTML, &answer.Status, &answer.EditedAt, &answer.CreatedAt)

	if err != nil {
		logger.Error("Failed to edit comment answer: %v", err)
		return nil, err
	}

	var user models.ProfilePublic
	var avatarUpdatedAt time.Time
	if err := database.DB.QueryRow(dbCtx, `SELECT display_name, avatar_seed, has_custom_avatar, avatar_updated_at FROM users WHERE id = $1`, userID).Scan(&user.DisplayName, &user.AvatarSeed, &user.HasCustomAvatar, &avatarUpdatedAt); err != nil {
		logger.Warn("Failed to fetch user data for edited answer: %v", err)
	}
	answer.UserDisplayName = user.DisplayName
	answer.UserAvatarSeed = user.AvatarSeed
	answer.UserHasCustomAvatar = user.HasCustomAvatar
	answer.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()

	go sendEditedAnswerToTelegram(context.Background(), &answer, oldContentHTML)

	recordCommentTime(userID)

	logger.Info("Comment answer %s edited by user %s", answer.ID, userID)
	return &answer, nil
}

func DeleteCommentAnswer(ctx context.Context, answerID, userID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ownerID string
	var status string
	err := database.DB.QueryRow(dbCtx,
		`SELECT user_id, status FROM comment_answers WHERE id = $1`,
		answerID,
	).Scan(&ownerID, &status)
	if err != nil {
		return ErrCommentAnswerNotFound
	}

	if ownerID != userID {
		return ErrNotAnswerAuthor
	}

	if status != "approved" {
		return ErrCannotDeleteAnswer
	}

	var id string
	err = database.DB.QueryRow(dbCtx, queryCommentAnswersUpdateStatus, "deleted", answerID).Scan(&id)
	if err != nil {
		return err
	}

	logger.Info("Comment answer %s deleted by user %s", answerID, userID)
	return nil
}

func UpdateCommentAnswerStatus(ctx context.Context, answerID, status string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var id string
	err := database.DB.QueryRow(dbCtx, queryCommentAnswersUpdateStatus, status, answerID).Scan(&id)
	if err != nil {
		logger.Error("Failed to update comment answer status: %v", err)
		return err
	}

	logger.Info("Comment answer %s status updated to %s", answerID, status)
	return nil
}

func sendCommentToTelegram(ctx context.Context, comment *models.Comment) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var novelID, novelTitle, chapterID, chapterTitle string
	var chapterNum int
	err := database.DB.QueryRow(dbCtx, queryCommentsTelegramInfo, comment.ChapterID).Scan(
		&novelID, &novelTitle, &chapterID, &chapterNum, &chapterTitle,
	)
	if err != nil {
		logger.Warn("Failed to fetch novel/chapter info for comment %s: %v", comment.ID, err)
		chapterID = comment.ChapterID
	}

	text := buildCommentTelegramText(
		novelID, novelTitle, chapterID, chapterNum, chapterTitle,
		comment.UserDisplayName, comment.ContentHTML,
	)

	sendTelegramMessage(ctx, "comment", comment.ID, text, "approve", "reject", queryCommentsSetTelegramMessageID)
}

func sendEditedCommentToTelegram(ctx context.Context, comment *models.Comment, oldContentHTML string) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var novelID, novelTitle, chapterID, chapterTitle string
	var chapterNum int
	err := database.DB.QueryRow(dbCtx, queryCommentsTelegramInfo, comment.ChapterID).Scan(
		&novelID, &novelTitle, &chapterID, &chapterNum, &chapterTitle,
	)
	if err != nil {
		logger.Warn("Failed to fetch novel/chapter info for edited comment %s: %v", comment.ID, err)
		chapterID = comment.ChapterID
	}

	text := buildEditedCommentTelegramText(
		novelID, novelTitle, chapterID, chapterNum, chapterTitle,
		comment.UserDisplayName, oldContentHTML, comment.ContentHTML,
	)

	sendTelegramMessage(ctx, "comment", comment.ID, text, "approve", "reject", queryCommentsSetTelegramMessageID)
}

func sendAnswerToTelegram(ctx context.Context, answer *models.CommentAnswer) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var novelID, novelTitle, chapterID, chapterTitle, parentAuthorName, parentContentHTML string
	var chapterNum int
	err := database.DB.QueryRow(dbCtx, queryCommentAnswersTelegramInfo, answer.CommentID).Scan(
		&novelID, &novelTitle, &chapterID, &chapterNum, &chapterTitle,
		&parentAuthorName, &parentContentHTML,
	)
	if err != nil {
		logger.Warn("Failed to fetch parent comment info for answer %s: %v", answer.ID, err)
	}

	text := buildAnswerTelegramText(
		novelID, novelTitle, chapterID, chapterNum, chapterTitle,
		parentAuthorName, parentContentHTML,
		answer.UserDisplayName, answer.ContentHTML,
	)

	sendTelegramMessage(ctx, "answer", answer.ID, text, "approve_answer", "reject_answer", queryCommentAnswersSetTelegramMessageID)
}

func sendEditedAnswerToTelegram(ctx context.Context, answer *models.CommentAnswer, oldContentHTML string) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var novelID, novelTitle, chapterID, chapterTitle, parentAuthorName, parentContentHTML string
	var chapterNum int
	err := database.DB.QueryRow(dbCtx, queryCommentAnswersTelegramInfo, answer.CommentID).Scan(
		&novelID, &novelTitle, &chapterID, &chapterNum, &chapterTitle,
		&parentAuthorName, &parentContentHTML,
	)
	if err != nil {
		logger.Warn("Failed to fetch parent comment info for edited answer %s: %v", answer.ID, err)
	}

	text := buildEditedAnswerTelegramText(
		novelID, novelTitle, chapterID, chapterNum, chapterTitle,
		parentAuthorName, parentContentHTML,
		answer.UserDisplayName, oldContentHTML, answer.ContentHTML,
	)

	sendTelegramMessage(ctx, "answer", answer.ID, text, "approve_answer", "reject_answer", queryCommentAnswersSetTelegramMessageID)
}

const telegramBaseURL = "https://kappalib.rip"

func formatTelegramChapterTitleInTable(chapterNum int, chapterTitle string) string {
	title := chapterTitle
	if title == "" || title == "Без названия" {
		title = "Без названия"
	}
	return fmt.Sprintf("%d. %s", chapterNum, title)
}

func buildTelegramMetadataTable(novelURL, displayNovelTitle, chapterURL string, chapterNum int, chapterTitle, authorName, replyToUser string) string {
	chapterText := formatTelegramChapterTitleInTable(chapterNum, chapterTitle)
	var sb strings.Builder
	sb.WriteString("<details><summary>Информация</summary><table>\n")
	fmt.Fprintf(&sb, "<tr><td>Новелла</td><td><a href=\"%s\">%s</a></td></tr>\n", stdhtml.EscapeString(novelURL), stdhtml.EscapeString(displayNovelTitle))
	fmt.Fprintf(&sb, "<tr><td>Глава</td><td><a href=\"%s\">%s</a></td></tr>\n", stdhtml.EscapeString(chapterURL), stdhtml.EscapeString(chapterText))
	fmt.Fprintf(&sb, "<tr><td>Автор</td><td>%s</td></tr>\n", stdhtml.EscapeString(authorName))
	if replyToUser != "" {
		fmt.Fprintf(&sb, "<tr><td>Кому</td><td>%s</td></tr>\n", stdhtml.EscapeString(replyToUser))
	}
	sb.WriteString("</table></details>")
	return sb.String()
}

func resolveDisplayTitle(novelTitle string) string {
	if novelTitle == "" {
		return "Страница новеллы"
	}
	return novelTitle
}

func resolveNovelURL(novelID string) string {
	if novelID != "" {
		return fmt.Sprintf("%s/%s", telegramBaseURL, novelID)
	}
	return telegramBaseURL
}

func resolveChapterURL(novelID, chapterID string) string {
	if novelID != "" && chapterID != "" {
		return fmt.Sprintf("%s/%s/chapter/%s", telegramBaseURL, novelID, chapterID)
	}
	if chapterID != "" {
		return fmt.Sprintf("%s/chapter/%s", telegramBaseURL, chapterID)
	}
	return telegramBaseURL
}

func truncateTelegramText(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}

func buildCommentTelegramText(novelID, novelTitle, chapterID string, chapterNum int, chapterTitle, authorName, contentHTML string) string {
	metadataTable := buildTelegramMetadataTable(resolveNovelURL(novelID), resolveDisplayTitle(novelTitle), resolveChapterURL(novelID, chapterID), chapterNum, chapterTitle, authorName, "")
	formattedContent := htmlToTelegramHTML(contentHTML)

	var sb strings.Builder
	sb.WriteString("<p>💬 Новый комментарий</p>\n")
	sb.WriteString(metadataTable)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "<details open><summary>Текст</summary>%s</details>", formattedContent)

	return truncateTelegramText(sb.String(), 30000)
}

func buildEditedCommentTelegramText(novelID, novelTitle, chapterID string, chapterNum int, chapterTitle, authorName, oldContentHTML, newContentHTML string) string {
	metadataTable := buildTelegramMetadataTable(resolveNovelURL(novelID), resolveDisplayTitle(novelTitle), resolveChapterURL(novelID, chapterID), chapterNum, chapterTitle, authorName, "")
	formattedOldContent := htmlToTelegramHTML(oldContentHTML)
	formattedNewContent := htmlToTelegramHTML(newContentHTML)

	var sb strings.Builder
	sb.WriteString("<p>📝 Новая редакция комментария</p>\n")
	sb.WriteString(metadataTable)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "<details><summary>Старая версия</summary>%s</details>\n", formattedOldContent)
	fmt.Fprintf(&sb, "<details open><summary>Текст</summary>%s</details>", formattedNewContent)

	return truncateTelegramText(sb.String(), 30000)
}

func buildAnswerTelegramText(
	novelID, novelTitle, chapterID string, chapterNum int, chapterTitle string,
	parentAuthorName, parentContentHTML string,
	answerAuthorName, answerContentHTML string,
) string {
	metadataTable := buildTelegramMetadataTable(resolveNovelURL(novelID), resolveDisplayTitle(novelTitle), resolveChapterURL(novelID, chapterID), chapterNum, chapterTitle, answerAuthorName, parentAuthorName)
	formattedParentContent := htmlToTelegramHTML(parentContentHTML)
	formattedAnswerContent := htmlToTelegramHTML(answerContentHTML)

	var sb strings.Builder
	sb.WriteString("<p>💬 Ответ на комментарий</p>\n")
	sb.WriteString(metadataTable)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "<details><summary>Комментарий</summary>%s</details>\n", formattedParentContent)
	fmt.Fprintf(&sb, "<details open><summary>Ответ</summary>%s</details>", formattedAnswerContent)

	return truncateTelegramText(sb.String(), 30000)
}

func buildEditedAnswerTelegramText(
	novelID, novelTitle, chapterID string, chapterNum int, chapterTitle string,
	parentAuthorName, parentContentHTML string,
	answerAuthorName, oldContentHTML, newContentHTML string,
) string {
	metadataTable := buildTelegramMetadataTable(resolveNovelURL(novelID), resolveDisplayTitle(novelTitle), resolveChapterURL(novelID, chapterID), chapterNum, chapterTitle, answerAuthorName, parentAuthorName)
	formattedParentContent := htmlToTelegramHTML(parentContentHTML)
	formattedOldContent := htmlToTelegramHTML(oldContentHTML)
	formattedNewContent := htmlToTelegramHTML(newContentHTML)

	var sb strings.Builder
	sb.WriteString("<p>📝 Новая редакция ответа</p>\n")
	sb.WriteString(metadataTable)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "<details><summary>Комментарий</summary>%s</details>\n", formattedParentContent)
	fmt.Fprintf(&sb, "<details><summary>Старая версия</summary>%s</details>\n", formattedOldContent)
	fmt.Fprintf(&sb, "<details open><summary>Ответ</summary>%s</details>", formattedNewContent)

	return truncateTelegramText(sb.String(), 30000)
}

func sendTelegramMessage(ctx context.Context, kind, id, text, approveAction, rejectAction, setMessageIDQuery string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if telegramBotToken == "" || telegramChatID == "" {
		logger.Warn("Telegram credentials not set, skipping notification")
		return
	}

	entityID := id
	if len(entityID) > 50 {
		entityID = entityID[:50]
		logger.Warn("%s ID truncated for Telegram callback: %s", kind, id)
	}

	keyboard := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "✅ Подтвердить", "callback_data": fmt.Sprintf("%s:%s", approveAction, entityID)},
				{"text": "❌ Отклонить", "callback_data": fmt.Sprintf("%s:%s", rejectAction, entityID)},
			},
		},
	}

	richPayload := map[string]any{
		"chat_id": telegramChatID,
		"rich_message": map[string]string{
			"html": text,
		},
		"reply_markup": keyboard,
	}
	richJSON, _ := json.Marshal(richPayload)
	richURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendRichMessage", telegramBotToken)

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}

	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", richURL, bytes.NewReader(richJSON))
		if err != nil {
			logger.Error("Failed to create telegram request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := telegramClient.Do(req)
		if err != nil {
			logger.Warn("Failed to send telegram rich message (attempt %d/3): %v", attempt+1, err)
			lastErr = err
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			_ = resp.Body.Close()
			logger.Warn("Failed to decode telegram response (attempt %d/3): %v", attempt+1, err)
			lastErr = err
			continue
		}
		_ = resp.Body.Close()

		if result.OK {
			if setMessageIDQuery != "" && result.Result.MessageID > 0 {
				if _, dbErr := database.DB.Exec(ctx, setMessageIDQuery, result.Result.MessageID, id); dbErr != nil {
					logger.Warn("Failed to update telegram_message_id: %v", dbErr)
				}
			}
			return
		}

		logger.Warn("Telegram API returned error (attempt %d/3): ok=false", attempt+1)
		lastErr = fmt.Errorf("telegram API returned ok=false")
	}

	logger.Error("Failed to send telegram message after 3 attempts: %v", lastErr)
}

func htmlToTelegramHTML(rawHTML string) string {
	rawHTML = strings.TrimSpace(rawHTML)
	if rawHTML == "" {
		return "[без текста]"
	}

	tokenizer := nethtml.NewTokenizer(strings.NewReader(rawHTML))
	var buf strings.Builder
	var openTags []string
	var inPre bool

	for {
		tt := tokenizer.Next()
		if tt == nethtml.ErrorToken {
			break
		}

		token := tokenizer.Token()
		switch tt {
		case nethtml.TextToken:
			buf.WriteString(stdhtml.EscapeString(token.Data))

		case nethtml.StartTagToken:
			switch token.Data {
			case "p":
				buf.WriteString("<p>")
				openTags = append(openTags, "p")
			case "br":
				buf.WriteString("<br/>")
			case "ul", "ol":
				buf.WriteString("<")
				buf.WriteString(token.Data)
				buf.WriteString(">")
				openTags = append(openTags, token.Data)
			case "li":
				buf.WriteString("<li>")
				openTags = append(openTags, "li")
			case "h1", "h2", "h3", "h4", "h5", "h6":
				if buf.Len() > 0 && !strings.HasSuffix(buf.String(), "\n") {
					buf.WriteString("\n")
				}
				buf.WriteString("<")
				buf.WriteString(token.Data)
				buf.WriteString(">")
				openTags = append(openTags, token.Data)
			case "strong", "b":
				buf.WriteString("<b>")
				openTags = append(openTags, "b")
			case "em", "i":
				buf.WriteString("<i>")
				openTags = append(openTags, "i")
			case "del", "s", "strike":
				buf.WriteString("<s>")
				openTags = append(openTags, "s")
			case "u", "ins":
				buf.WriteString("<u>")
				openTags = append(openTags, "u")
			case "code":
				if !inPre {
					buf.WriteString("<code>")
					openTags = append(openTags, "code")
				}
			case "pre":
				inPre = true
				buf.WriteString("<pre>")
				openTags = append(openTags, "pre")
			case "blockquote":
				if buf.Len() > 0 && !strings.HasSuffix(buf.String(), "\n") {
					buf.WriteString("\n")
				}
				buf.WriteString("<blockquote>")
				openTags = append(openTags, "blockquote")
			case "span":
				for _, attr := range token.Attr {
					if attr.Key == "class" && attr.Val == "spoiler" {
						buf.WriteString("<tg-spoiler>")
						openTags = append(openTags, "tg-spoiler")
						break
					}
				}
			case "a":
				var href string
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						href = attr.Val
						break
					}
				}
				if href != "" {
					fmt.Fprintf(&buf, `<a href="%s">`, stdhtml.EscapeString(href))
					openTags = append(openTags, "a")
				}
			case "img":
				var src string
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						src = attr.Val
						break
					}
				}
				if src != "" {
					fmt.Fprintf(&buf, `<img src="%s"/>`, stdhtml.EscapeString(src))
				}
			case "hr":
				buf.WriteString("\n<hr/>\n")
			}

		case nethtml.EndTagToken:
			switch token.Data {
			case "p":
				if popTelegramTag(&openTags, "p") {
					buf.WriteString("</p>")
				}
			case "ul", "ol":
				if popTelegramTag(&openTags, token.Data) {
					buf.WriteString("</")
					buf.WriteString(token.Data)
					buf.WriteString(">")
				}
			case "li":
				if popTelegramTag(&openTags, "li") {
					buf.WriteString("</li>")
				}
			case "h1", "h2", "h3", "h4", "h5", "h6":
				if popTelegramTag(&openTags, token.Data) {
					buf.WriteString("</")
					buf.WriteString(token.Data)
					buf.WriteString(">")
				}
				if !strings.HasSuffix(buf.String(), "\n") {
					buf.WriteString("\n")
				}
			case "strong", "b":
				if popTelegramTag(&openTags, "b") {
					buf.WriteString("</b>")
				}
			case "em", "i":
				if popTelegramTag(&openTags, "i") {
					buf.WriteString("</i>")
				}
			case "del", "s", "strike":
				if popTelegramTag(&openTags, "s") {
					buf.WriteString("</s>")
				}
			case "u", "ins":
				if popTelegramTag(&openTags, "u") {
					buf.WriteString("</u>")
				}
			case "code":
				if !inPre && popTelegramTag(&openTags, "code") {
					buf.WriteString("</code>")
				}
			case "pre":
				inPre = false
				if popTelegramTag(&openTags, "pre") {
					buf.WriteString("</pre>")
				}
			case "blockquote":
				if popTelegramTag(&openTags, "blockquote") {
					buf.WriteString("</blockquote>")
				}
			case "span":
				if popTelegramTag(&openTags, "tg-spoiler") {
					buf.WriteString("</tg-spoiler>")
				}
			case "a":
				if popTelegramTag(&openTags, "a") {
					buf.WriteString("</a>")
				}
			}

		case nethtml.SelfClosingTagToken:
			switch token.Data {
			case "br":
				buf.WriteString("<br/>")
			case "img":
				var src string
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						src = attr.Val
						break
					}
				}
				if src != "" {
					fmt.Fprintf(&buf, `<img src="%s"/>`, stdhtml.EscapeString(src))
				}
			case "hr":
				buf.WriteString("\n<hr/>\n")
			}
		}
	}

	for _, openTag := range slices.Backward(openTags) {
		switch openTag {
		case "p":
			buf.WriteString("</p>")
		case "ul", "ol", "li":
			buf.WriteString("</")
			buf.WriteString(openTag)
			buf.WriteString(">")
		case "h1", "h2", "h3", "h4", "h5", "h6":
			buf.WriteString("</")
			buf.WriteString(openTag)
			buf.WriteString(">")
		case "b":
			buf.WriteString("</b>")
		case "i":
			buf.WriteString("</i>")
		case "blockquote":
			buf.WriteString("</blockquote>")
		case "s":
			buf.WriteString("</s>")
		case "u":
			buf.WriteString("</u>")
		case "code":
			buf.WriteString("</code>")
		case "pre":
			buf.WriteString("</pre>")
		case "tg-spoiler":
			buf.WriteString("</tg-spoiler>")
		case "a":
			buf.WriteString("</a>")
		}
	}

	res := strings.TrimSpace(buf.String())
	if res == "" {
		return "[без текста]"
	}
	return res
}

func popTelegramTag(tags *[]string, target string) bool {
	for i := range slices.Backward(*tags) {
		if (*tags)[i] == target {
			*tags = append((*tags)[:i], (*tags)[i+1:]...)
			return true
		}
	}
	return false
}

var imageUploadLimiter = struct {
	sync.Mutex
	lastUpload map[string]time.Time
}{
	lastUpload: make(map[string]time.Time),
}

const imageUploadCooldown = 3 * time.Second

func acquireImageUploadSlot(userID string) bool {
	imageUploadLimiter.Lock()
	defer imageUploadLimiter.Unlock()
	if last, exists := imageUploadLimiter.lastUpload[userID]; exists {
		if time.Since(last) < imageUploadCooldown {
			return false
		}
	}
	imageUploadLimiter.lastUpload[userID] = time.Now()
	return true
}

func detectImageFormat(data []byte) (string, string, error) {
	if len(data) < 4 {
		return "", "", ErrUnsupportedFormat
	}
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg", "jpg", nil
	}
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png", "png", nil
	}
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "image/gif", "gif", nil
	}
	return "", "", ErrUnsupportedFormat
}

func hashImageData(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

func UploadCommentImage(ctx context.Context, userID string, imageData []byte) (string, error) {
	if minioClient == nil {
		return "", ErrS3NotConfigured
	}

	if !acquireImageUploadSlot(userID) {
		return "", ErrRateLimitExceeded
	}

	contentType, ext, err := detectImageFormat(imageData)
	if err != nil {
		return "", err
	}

	hash := hashImageData(imageData)
	key := fmt.Sprintf("comments/%s_%s.%s", userID, hash, ext)

	s3Ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_, statErr := minioClient.StatObject(s3Ctx, s3Bucket, key, minio.StatObjectOptions{})
	if statErr == nil {
		s3PublicURL := os.Getenv("S3_PUBLIC_URL")
		logger.Debug("Comment image already exists, skipping upload: %s", key)
		return fmt.Sprintf("%s/%s", s3PublicURL, key), nil
	}

	reader := bytes.NewReader(imageData)
	_, err = minioClient.PutObject(s3Ctx, s3Bucket, key, reader, int64(len(imageData)), minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		logger.Error("S3 upload failed for comment image (user %s, key %s): %v", userID, key, err)
		return "", fmt.Errorf("s3 upload failed: %w", err)
	}

	logger.Info("Comment image uploaded: %s by user %s (%d bytes)", key, userID, len(imageData))

	s3PublicURL := os.Getenv("S3_PUBLIC_URL")
	return fmt.Sprintf("%s/%s", s3PublicURL, key), nil
}

func GetTelegramWebhookSecret() string {
	return telegramWebhookSecret
}

func GetTelegramChatID() string {
	return telegramChatID
}

func SetTelegramWebhookSecret(secret string) {
	telegramWebhookSecret = secret
}

func SetTelegramChatID(chatID string) {
	telegramChatID = chatID
}

func GetUserCommentStats(ctx context.Context, userID string) (*models.UserCommentStats, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var baseRating, baseReplies int
	if err := database.DB.QueryRow(dbCtx, queryCommentStatsBase, userID).Scan(&baseRating, &baseReplies); err != nil {
		logger.Error("Failed to get comment stats base: %v", err)
		return nil, err
	}

	rows, err := database.DB.Query(dbCtx, queryCommentStatsDaily, userID)
	if err != nil {
		logger.Error("Failed to get user comment stats: %v", err)
		return nil, err
	}
	defer rows.Close()

	days := make([]models.CommentStatDay, 0, 30)
	cumRating, cumReplies := baseRating, baseReplies

	for rows.Next() {
		var d models.CommentStatDay
		var dailyRating, dailyReplies int
		if err := rows.Scan(&d.Day, &dailyRating, &dailyReplies); err != nil {
			continue
		}
		cumRating += dailyRating
		cumReplies += dailyReplies
		d.Rating = cumRating
		d.Replies = cumReplies
		days = append(days, d)
	}

	return &models.UserCommentStats{
		Days:    days,
		Rating:  cumRating,
		Replies: cumReplies,
	}, nil
}

func chapterExists(ctx context.Context, chapterID string) bool {
	var exists bool
	err := database.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chapters WHERE id = $1)`,
		chapterID,
	).Scan(&exists)
	return err == nil && exists
}

func DeleteTelegramMessage(chatID int64, messageID int64) error {
	if telegramBotToken == "" {
		return fmt.Errorf("telegram bot token not set")
	}

	apiURL := fmt.Sprintf(
		"https://api.telegram.org/bot%s/deleteMessage",
		telegramBotToken,
	)

	data := url.Values{
		"chat_id":    {fmt.Sprintf("%d", chatID)},
		"message_id": {fmt.Sprintf("%d", messageID)},
	}

	resp, err := telegramClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		OK bool `json:"ok"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf("telegram deleteMessage failed")
	}

	return nil
}
