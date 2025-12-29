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
	database.DB.QueryRow(dbCtx, `SELECT display_name, avatar_seed FROM users WHERE id = $1`, profileID).Scan(&user.DisplayName, &user.AvatarSeed)
	comment.UserDisplayName = user.DisplayName
	comment.UserAvatarSeed = user.AvatarSeed

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
		if err := rows.Scan(&c.ID, &c.ChapterID, &c.UserID, &c.ContentHTML, &c.Status, &c.CreatedAt, &c.UserDisplayName, &c.UserAvatarSeed); err != nil {
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

	text := fmt.Sprintf(
		"💬 *Новый комментарий*\n\n"+
			"👤 Автор: %s\n"+
			"📖 Глава: `%s`\n\n"+
			"📝 Текст:\n%s",
		escapeMarkdown(comment.UserDisplayName),
		comment.ChapterID,
		escapeMarkdown(stripHTMLTags(comment.ContentHTML)),
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
		"parse_mode":   {"Markdown"},
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

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}

func GetTelegramWebhookSecret() string {
	return telegramWebhookSecret
}

func UpdateTelegramMessage(messageID int64, newText string) error {
	if telegramBotToken == "" || telegramChatID == "" {
		return fmt.Errorf("telegram credentials not set")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", telegramBotToken)

	data := url.Values{
		"chat_id":    {telegramChatID},
		"message_id": {fmt.Sprintf("%d", messageID)},
		"text":       {newText},
		"parse_mode": {"Markdown"},
	}

	resp, err := telegramClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

func chapterExists(ctx context.Context, chapterID string) bool {
	var exists bool
	err := database.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chapters WHERE id = $1)`,
		chapterID,
	).Scan(&exists)
	return err == nil && exists
}
