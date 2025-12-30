package data

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
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

//go:embed sql/comments_get_approved.sql
var queryCommentsGetApproved string

//go:embed sql/comments_count_approved.sql
var queryCommentsCountApproved string

//go:embed sql/comments_get_by_id.sql
var queryCommentsGetByID string

//go:embed sql/comments_update_status.sql
var queryCommentsUpdateStatus string

//go:embed sql/comments_set_telegram_message_id.sql
var queryCommentsSetTelegramMessageID string

var (
	commentsTurnstileSecret = os.Getenv("TURNSTILE_COMMENTS_SECRET")
	telegramBotToken        = os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID          = os.Getenv("TELEGRAM_CHAT_ID")
	telegramWebhookSecret   = os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	markdownPolicy          *bluemonday.Policy
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
	markdownPolicy.AllowAttrs("href").OnElements("a")
	markdownPolicy.AllowURLSchemes("http", "https")
	markdownPolicy.AllowImages()
	markdownPolicy.AllowAttrs("src", "alt", "title").OnElements("img")
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
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
	defer resp.Body.Close()

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
	return strings.TrimSpace(string(safe))
}

func CreateComment(ctx context.Context, profileID, secretToken string, input models.CreateCommentInput) (*models.Comment, error) {
	if len(input.Content) == 0 || len(input.Content) > 1000 {
		return nil, fmt.Errorf("invalid content length")
	}

	if !checkCommentRateLimit(profileID) {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	if !verifyCommentsTurnstile(input.TurnstileToken) {
		return nil, fmt.Errorf("captcha verification failed")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !chapterExists(dbCtx, input.ChapterID) {
		return nil, fmt.Errorf("chapter not found")
	}

	if !verifySecretToken(dbCtx, profileID, secretToken) {
		return nil, fmt.Errorf("invalid secret token")
	}

	contentHTML := renderMarkdown(input.Content)

	var comment models.Comment
	err := database.DB.QueryRow(dbCtx, queryCommentsCreate,
		input.ChapterID, profileID, contentHTML,
	).Scan(&comment.ID, &comment.ChapterID, &comment.UserID, &comment.ContentHTML, &comment.Status, &comment.CreatedAt)

	if err != nil {
		logger.Error("Failed to create comment: %v", err)
		return nil, err
	}

	var user models.ProfilePublic
	database.DB.QueryRow(dbCtx, `SELECT display_name, avatar_seed, has_custom_avatar FROM users WHERE id = $1`, profileID).Scan(&user.DisplayName, &user.AvatarSeed, &user.HasCustomAvatar)
	comment.UserDisplayName = user.DisplayName
	comment.UserAvatarSeed = user.AvatarSeed
	comment.UserHasCustomAvatar = user.HasCustomAvatar

	go sendCommentToTelegram(context.Background(), &comment)

	recordCommentTime(profileID)

	logger.Info("Comment created: %s by user %s", comment.ID, profileID)
	return &comment, nil
}

func GetApprovedComments(ctx context.Context, chapterID string, page int) (*models.CommentsPage, error) {
	pageSize := 12
	offset := (page - 1) * pageSize

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var totalCount int
	if err := database.DB.QueryRow(dbCtx, queryCommentsCountApproved, chapterID).Scan(&totalCount); err != nil {
		logger.Error("Failed to count comments: %v", err)
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

	rows, err := database.DB.Query(dbCtx, queryCommentsGetApproved, chapterID, pageSize, offset)
	if err != nil {
		logger.Error("Failed to get comments: %v", err)
		return nil, err
	}
	defer rows.Close()

	comments := make([]models.Comment, 0)
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status, &c.CreatedAt, &c.UserDisplayName, &c.UserAvatarSeed, &c.UserHasCustomAvatar); err != nil {
			logger.Warn("Comment row scan error: %v", err)
			continue
		}
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

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		logger.Error("Failed to create telegram request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := telegramClient.Do(req)
	if err != nil {
		logger.Error("Failed to send telegram message: %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("Failed to decode telegram response: %v", err)
		return
	}

	if result.OK {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		database.DB.Exec(ctx, queryCommentsSetTelegramMessageID, result.Result.MessageID, comment.ID)
	}
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

func GetTelegramWebhookSecret() string {
	return telegramWebhookSecret
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
	defer resp.Body.Close()

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
