package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/ch1kulya/logger"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

//go:embed sql/comments_create.sql
var queryCommentsCreate string

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

//go:embed sql/comment_answers_update_status.sql
var queryCommentAnswersUpdateStatus string

//go:embed sql/comment_answers_set_telegram_message_id.sql
var queryCommentAnswersSetTelegramMessageID string

//go:embed sql/user_comment_threads.sql
var queryUserCommentThreads string

//go:embed sql/comment_stats_base.sql
var queryCommentStatsBase string

//go:embed sql/comment_stats_daily.sql
var queryCommentStatsDaily string

var (
	commentsTurnstileSecret = os.Getenv("TURNSTILE_COMMENTS_SECRET")
	telegramBotToken        = os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID          = os.Getenv("TELEGRAM_CHAT_ID")
	telegramWebhookSecret   = os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	markdownPolicy          *bluemonday.Policy
	spoilerRegex            = regexp.MustCompile(`(?s)\|\|(.+?)\|\|`)
	telegramClient          = &http.Client{
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

func renderMarkdown(content string) string {
	unsafe := blackfriday.Run([]byte(content),
		blackfriday.WithExtensions(blackfriday.CommonExtensions&^blackfriday.Tables&^blackfriday.FencedCode),
	)
	safe := markdownPolicy.SanitizeBytes(unsafe)
	result := spoilerRegex.ReplaceAllString(string(safe), `<span class="spoiler">$1</span>`)
	result = strings.ReplaceAll(result, "<img src=", `<img loading="lazy" src=`)
	return strings.TrimSpace(result)
}

func CreateComment(ctx context.Context, userID string, input models.CreateCommentInput) (*models.Comment, error) {
	if len(input.Content) == 0 || len(input.Content) > 3000 {
		return nil, ErrInvalidContentLength
	}

	if !checkCommentRateLimit(userID) {
		return nil, ErrRateLimitExceeded
	}

	if !verifyCommentsTurnstile(input.TurnstileToken) {
		return nil, ErrCaptchaFailed
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
	).Scan(&comment.ID, &comment.ChapterID, &comment.UserID, &comment.ContentHTML, &comment.Status, &comment.CreatedAt)

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

func GetVisibleComments(ctx context.Context, chapterID, userID string, page int) (*models.CommentsPage, error) {
	pageSize := 12
	offset := (page - 1) * pageSize

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var totalCount int
	err := database.DB.QueryRow(dbCtx, `
        SELECT COUNT(*)
        FROM comments c
        WHERE c.chapter_id = $1
          AND c.status != 'deleted'
          AND (
            c.status = 'approved'
            OR (c.status IN ('pending', 'rejected') AND c.user_id = $2)
          )
    `, chapterID, userID).Scan(&totalCount)
	if err != nil {
		logger.Error("Failed to count visible comments: %v", err)
		return nil, err
	}

	if totalCount == 0 {
		return &models.CommentsPage{
			Comments:   []models.Comment{},
			Page:       page,
			PageSize:   pageSize,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	rows, err := database.DB.Query(dbCtx, `
        SELECT
            c.id, c.chapter_id, c.user_id, c.content_html, c.status, c.created_at,
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
        ORDER BY c.created_at DESC
        LIMIT $3 OFFSET $4
    `, chapterID, userID, pageSize, offset)
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
			&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status,
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
				ca.id, ca.comment_id, ca.user_id, ca.content_html, ca.status, ca.created_at,
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
					&a.ID, &a.CommentID, &a.UserID, &a.ContentHTML, &a.Status,
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

	totalPages := (totalCount + pageSize - 1) / pageSize
	return &models.CommentsPage{
		Comments:   comments,
		Page:       page,
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
			c.id, c.chapter_id, c.user_id, c.content_html, c.status, c.created_at,
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
			&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status,
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
			ca.id, ca.comment_id, ca.user_id, ca.content_html, ca.status, ca.created_at,
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
		defer answerRows.Close()
		for answerRows.Next() {
			var a models.CommentAnswer
			var avatarUpdatedAt time.Time
			if err := answerRows.Scan(
				&a.ID, &a.CommentID, &a.UserID, &a.ContentHTML, &a.Status,
				&a.CreatedAt, &a.UserDisplayName, &a.UserAvatarSeed,
				&a.UserHasCustomAvatar, &avatarUpdatedAt,
			); err != nil {
				logger.Warn("Answer row scan error: %v", err)
				continue
			}
			a.UserAvatarUpdatedAt = avatarUpdatedAt.Unix()
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
		`SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1 AND status = 'approved')`,
		commentID,
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

	logger.Info("Comment %s deleted by user %s", commentID, userID)
	return nil
}

func GetCommentByID(ctx context.Context, commentID string) (*models.Comment, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var c models.Comment
	err := database.DB.QueryRow(dbCtx, queryCommentsGetByID, commentID).Scan(
		&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status, &c.TelegramMessageID, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func CreateCommentAnswer(ctx context.Context, userID string, input models.CreateCommentAnswerInput) (*models.CommentAnswer, error) {
	if len(input.Content) == 0 || len(input.Content) > 500 {
		return nil, ErrInvalidAnswerLength
	}

	if !checkCommentRateLimit(userID) {
		return nil, ErrRateLimitExceeded
	}

	if !verifyCommentsTurnstile(input.TurnstileToken) {
		return nil, ErrCaptchaFailed
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var commentStatus string
	err := database.DB.QueryRow(dbCtx,
		`SELECT status FROM comments WHERE id = $1`,
		input.CommentID,
	).Scan(&commentStatus)
	if err != nil {
		return nil, ErrCommentNotFound
	}
	if commentStatus != "approved" {
		return nil, ErrCommentNotApproved
	}

	contentHTML := renderMarkdown(input.Content)

	var answer models.CommentAnswer
	err = database.DB.QueryRow(dbCtx, queryCommentAnswersCreate,
		input.CommentID, userID, contentHTML,
	).Scan(&answer.ID, &answer.CommentID, &answer.UserID, &answer.ContentHTML, &answer.Status, &answer.CreatedAt)

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
	sendToTelegram(ctx, "comment", comment.ID, comment.UserDisplayName, comment.ChapterID, comment.ContentHTML, "approve", "reject", queryCommentsSetTelegramMessageID)
}

func sendAnswerToTelegram(ctx context.Context, answer *models.CommentAnswer) {
	sendToTelegram(ctx, "answer", answer.ID, answer.UserDisplayName, answer.CommentID, answer.ContentHTML, "approve_answer", "reject_answer", queryCommentAnswersSetTelegramMessageID)
}

func sendToTelegram(ctx context.Context, kind, id, authorName, contextID, contentHTML, approveAction, rejectAction, setMessageIDQuery string) {
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

	contentForTelegram := htmlToTelegramHTML(contentHTML)

	var label, contextLabel string
	if kind == "comment" {
		label, contextLabel = "💬 <b>Новый комментарий</b>", "📖 Глава"
	} else {
		label, contextLabel = "💬 <b>Ответ на комментарий</b>", "🔗 Комментарий"
	}

	text := fmt.Sprintf(
		"%s\n\n"+
			"👤 Автор: %s\n"+
			"%s: <code>%s</code>\n\n"+
			"📝 Текст:\n%s",
		label, authorName, contextLabel, contextID, contentForTelegram,
	)

	if len(text) > 4000 {
		text = text[:4000] + "..."
	}

	keyboard := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "✅ Подтвердить", "callback_data": fmt.Sprintf("%s:%s", approveAction, entityID)},
				{"text": "❌ Отклонить", "callback_data": fmt.Sprintf("%s:%s", rejectAction, entityID)},
			},
		},
	}

	keyboardJSON, _ := json.Marshal(keyboard)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)

	urlData := url.Values{
		"chat_id":      {telegramChatID},
		"text":         {text},
		"parse_mode":   {"HTML"},
		"reply_markup": {string(keyboardJSON)},
	}

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

		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(urlData.Encode()))
		if err != nil {
			logger.Error("Failed to create telegram request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := telegramClient.Do(req)
		if err != nil {
			logger.Warn("Failed to send telegram message (attempt %d/3): %v", attempt+1, err)
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
			dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer dbCancel()
			if _, err := database.DB.Exec(dbCtx, setMessageIDQuery, result.Result.MessageID, id); err != nil {
				logger.Warn("Failed to set telegram message ID for %s %s: %v", kind, id, err)
			}
			return
		}

		logger.Warn("Telegram API returned error (attempt %d/3): ok=false", attempt+1)
		lastErr = fmt.Errorf("telegram API returned ok=false")
	}

	logger.Error("Failed to send telegram message after 3 attempts: %v", lastErr)
}

func htmlToTelegramHTML(html string) string {
	if html == "" {
		return "[без текста]"
	}
	result := html
	for _, h := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		result = strings.ReplaceAll(result, "<"+h+">", "<b>")
		result = strings.ReplaceAll(result, "</"+h+">", "</b>\n")
	}
	result = strings.ReplaceAll(result, "<p>", "")
	result = strings.ReplaceAll(result, "</p>", "\n\n")
	result = strings.ReplaceAll(result, "<br>", "\n")
	result = strings.ReplaceAll(result, "<br/>", "\n")
	result = strings.ReplaceAll(result, "<br />", "\n")
	result = strings.ReplaceAll(result, "<ul>", "")
	result = strings.ReplaceAll(result, "</ul>", "\n")
	result = strings.ReplaceAll(result, "<ol>", "")
	result = strings.ReplaceAll(result, "</ol>", "\n")
	result = strings.ReplaceAll(result, "<li>", "• ")
	result = strings.ReplaceAll(result, "</li>", "\n")
	result = strings.ReplaceAll(result, "<strong>", "<b>")
	result = strings.ReplaceAll(result, "</strong>", "</b>")
	result = strings.ReplaceAll(result, "<em>", "<i>")
	result = strings.ReplaceAll(result, "</em>", "</i>")
	result = strings.ReplaceAll(result, `<span class="spoiler">`, "<tg-spoiler>")
	result = strings.ReplaceAll(result, "</span>", "</tg-spoiler>")
	result = replaceImgTags(result)

	result = strings.TrimSpace(result)

	if result == "" {
		return "[без текста]"
	}

	return result
}

func replaceImgTags(html string) string {
	result := html
	for {
		start := strings.Index(result, "<img")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		end += start
		imgTag := result[start : end+1]
		src := ""
		if srcStart := strings.Index(imgTag, `src="`); srcStart != -1 {
			srcStart += 5
			if srcEnd := strings.Index(imgTag[srcStart:], `"`); srcEnd != -1 {
				src = imgTag[srcStart : srcStart+srcEnd]
			}
		}
		alt := "изображение"
		if altStart := strings.Index(imgTag, `alt="`); altStart != -1 {
			altStart += 5
			if altEnd := strings.Index(imgTag[altStart:], `"`); altEnd != -1 {
				if a := imgTag[altStart : altStart+altEnd]; a != "" {
					alt = a
				}
			}
		}
		replacement := "[🖼 " + alt + "]"
		if src != "" {
			replacement = `<a href="` + src + `">[🖼 ` + alt + `]</a>`
		}
		result = result[:start] + replacement + result[end+1:]
	}
	return result
}

var imageUploadLimiter = struct {
	sync.Mutex
	lastUpload map[string]time.Time
}{
	lastUpload: make(map[string]time.Time),
}

const imageUploadCooldown = 3 * time.Second

func checkImageUploadRateLimit(userID string) bool {
	imageUploadLimiter.Lock()
	defer imageUploadLimiter.Unlock()
	if last, exists := imageUploadLimiter.lastUpload[userID]; exists {
		if time.Since(last) < imageUploadCooldown {
			return false
		}
	}
	return true
}

func recordImageUploadTime(userID string) {
	imageUploadLimiter.Lock()
	defer imageUploadLimiter.Unlock()
	imageUploadLimiter.lastUpload[userID] = time.Now()
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

	if !checkImageUploadRateLimit(userID) {
		return "", ErrRateLimitExceeded
	}

	recordImageUploadTime(userID)

	contentType, ext, err := detectImageFormat(imageData)
	if err != nil {
		return "", err
	}

	hash := hashImageData(imageData)
	key := fmt.Sprintf("comments/%s_%s.%s", userID, hash, ext)

	_, statErr := minioClient.StatObject(ctx, s3Bucket, key, minio.StatObjectOptions{})
	if statErr == nil {
		s3PublicURL := os.Getenv("S3_PUBLIC_URL")
		logger.Debug("Comment image already exists, skipping upload: %s", key)
		return fmt.Sprintf("%s/%s", s3PublicURL, key), nil
	}

	reader := bytes.NewReader(imageData)
	_, err = minioClient.PutObject(ctx, s3Bucket, key, reader, int64(len(imageData)), minio.PutObjectOptions{
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
	database.DB.QueryRow(dbCtx, queryCommentStatsBase, userID).Scan(&baseRating, &baseReplies)

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
