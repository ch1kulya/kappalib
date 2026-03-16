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

//go:embed sql/comments_get_by_user.sql
var queryCommentsGetByUser string

//go:embed sql/comments_count_by_user.sql
var queryCommentsCountByUser string

//go:embed sql/comment_votes_upsert.sql
var queryCommentVotesUpsert string

//go:embed sql/comment_votes_delete.sql
var queryCommentVotesDelete string

//go:embed sql/comment_votes_score.sql
var queryCommentVotesScore string

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

	var totalCount int
	err := database.DB.QueryRow(dbCtx, queryCommentsCountByUser, userID).Scan(&totalCount)
	if err != nil {
		logger.Error("Failed to count user comments: %v", err)
		return nil, err
	}

	if totalCount == 0 {
		return &models.UserCommentsPage{
			Comments:   []models.UserComment{},
			Page:       page,
			PageSize:   pageSize,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	rows, err := database.DB.Query(dbCtx, queryCommentsGetByUser, userID, pageSize, offset)
	if err != nil {
		logger.Error("Failed to get user comments: %v", err)
		return nil, err
	}
	defer rows.Close()

	comments := make([]models.UserComment, 0)
	for rows.Next() {
		var c models.UserComment
		if err := rows.Scan(
			&c.ID, &c.ChapterID, &c.ContentHTML, &c.Status, &c.CreatedAt,
			&c.ChapterNum, &c.NovelID, &c.NovelTitle, &c.Score, &c.UserVote,
		); err != nil {
			logger.Warn("User comment row scan error: %v", err)
			continue
		}
		comments = append(comments, c)
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

func sendCommentToTelegram(ctx context.Context, comment *models.Comment) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if telegramBotToken == "" || telegramChatID == "" {
		logger.Warn("Telegram credentials not set, skipping notification")
		return
	}

	commentID := comment.ID
	if len(commentID) > 50 {
		commentID = commentID[:50]
		logger.Warn("Comment ID truncated for Telegram callback: %s", comment.ID)
	}

	contentForTelegram := htmlToTelegramHTML(comment.ContentHTML)

	text := fmt.Sprintf(
		"💬 <b>Новый комментарий</b>\n\n"+
			"👤 Автор: %s\n"+
			"📖 Глава: <code>%s</code>\n\n"+
			"📝 Текст:\n%s",
		comment.UserDisplayName,
		comment.ChapterID,
		contentForTelegram,
	)

	if len(text) > 4000 {
		text = text[:4000] + "..."
	}

	keyboard := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "✅ Подтвердить", "callback_data": fmt.Sprintf("approve:%s", commentID)},
				{"text": "❌ Отклонить", "callback_data": fmt.Sprintf("reject:%s", commentID)},
			},
		},
	}

	keyboardJSON, _ := json.Marshal(keyboard)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)

	data := url.Values{
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

		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
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
		defer func() { _ = resp.Body.Close() }()

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			logger.Warn("Failed to decode telegram response (attempt %d/3): %v", attempt+1, err)
			lastErr = err
			continue
		}

		if result.OK {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := database.DB.Exec(ctx, queryCommentsSetTelegramMessageID, result.Result.MessageID, comment.ID); err != nil {
				logger.Warn("Failed to set telegram message ID for comment %s: %v", comment.ID, err)
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
