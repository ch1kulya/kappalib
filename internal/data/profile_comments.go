package data

import (
	"context"
	_ "embed"
	"fmt"
	stdhtml "html"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
)

//go:embed sql/profile_comments_create.sql
var queryProfileCommentsCreate string

//go:embed sql/profile_comments_count.sql
var queryProfileCommentsCount string

//go:embed sql/profile_comments_position.sql
var queryProfileCommentsPosition string

//go:embed sql/profile_comments_list.sql
var queryProfileCommentsList string

//go:embed sql/users_profile_comments_seen_get.sql
var queryUsersProfileCommentsSeenGet string

//go:embed sql/users_profile_comments_seen_update.sql
var queryUsersProfileCommentsSeenUpdate string

var profileCommentQueries = commentTargetQueries{
	position: queryProfileCommentsPosition,
	count:    queryProfileCommentsCount,
	list:     queryProfileCommentsList,
}

func CreateProfileComment(ctx context.Context, userID string, input models.CreateProfileCommentInput) (*models.Comment, error) {
	if _, err := GetPublicProfile(ctx, input.ProfileID); err != nil {
		return nil, err
	}

	if err := validateSubmission(userID, input.Content, 3000, false, input.TurnstileToken, input.SmartCaptchaToken, input.IP); err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	contentHTML := renderMarkdown(input.Content)

	var comment models.Comment
	err := database.DB.QueryRow(dbCtx, queryProfileCommentsCreate,
		input.ProfileID, userID, contentHTML,
	).Scan(&comment.ID, &comment.UserID, &comment.ContentHTML, &comment.Status, &comment.EditedAt, &comment.CreatedAt, &comment.ProfileID)
	if err != nil {
		logger.Error("Failed to create profile comment: %v", err)
		return nil, err
	}

	comment.SetAuthor(loadCommentAuthor(dbCtx, userID))

	go sendCommentToTelegram(context.Background(), &comment)

	recordCommentTime(userID)

	logger.Info("Profile comment created: %s by user %s on profile %s", comment.ID, userID, input.ProfileID)
	return &comment, nil
}

func getProfileCommentsSeen(ctx context.Context, userID string) time.Time {
	var seenAt time.Time
	if err := database.DB.QueryRow(ctx, queryUsersProfileCommentsSeenGet, userID).Scan(&seenAt); err != nil {
		logger.Warn("Failed to get profile_comments_last_seen for user %s: %v", userID, err)
		return time.Now()
	}
	return seenAt
}

func MarkProfileCommentsSeen(ctx context.Context, userID string) error {
	_, err := database.DB.Exec(ctx, queryUsersProfileCommentsSeenUpdate, userID)
	return err
}

func GetVisibleProfileComments(ctx context.Context, profileID, viewerID string, page int, targetID string, seenBefore time.Time) (*models.CommentsPage, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var unseenSince *time.Time
	if viewerID != "" && viewerID == profileID {
		seenAt := earliestSeen(getProfileCommentsSeen(dbCtx, viewerID), seenBefore)
		unseenSince = &seenAt
	}

	result, err := getVisibleTargetComments(dbCtx, profileCommentQueries, profileID, viewerID, page, targetID, unseenSince)
	if err != nil {
		return nil, err
	}

	if result.TotalCount == 0 {
		if _, err := GetPublicProfile(dbCtx, profileID); err != nil {
			return nil, err
		}
	}

	result.SeenBefore = unseenSince
	return result, nil
}

func sendProfileCommentToTelegram(ctx context.Context, comment *models.Comment, oldContentHTML string, edited bool) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var profileName string
	if profile, err := GetPublicProfile(dbCtx, comment.ProfileID); err != nil {
		logger.Warn("Failed to fetch profile info for comment %s: %v", comment.ID, err)
	} else {
		profileName = profile.DisplayName
	}

	var text string
	if edited {
		text = buildEditedProfileCommentTelegramText(comment.ProfileID, profileName, comment.UserDisplayName, oldContentHTML, comment.ContentHTML)
	} else {
		text = buildProfileCommentTelegramText(comment.ProfileID, profileName, comment.UserDisplayName, comment.ContentHTML)
	}

	sendTelegramMessage(ctx, "comment", comment.ID, text, "approve", "reject", queryCommentsSetTelegramMessageID)
}

func resolveProfileURL(profileID string) string {
	if profileID != "" {
		return fmt.Sprintf("%s/%s", telegramBaseURL, profileID)
	}
	return telegramBaseURL
}

func resolveProfileName(profileName string) string {
	if profileName == "" {
		return "Профиль пользователя"
	}
	return profileName
}

func buildProfileTelegramMetadataTable(profileURL, profileName, authorName string) string {
	var sb strings.Builder
	sb.WriteString("<details><summary>Информация</summary><table>\n")
	fmt.Fprintf(&sb, "<tr><td>Профиль</td><td><a href=\"%s\">%s</a></td></tr>\n", stdhtml.EscapeString(profileURL), stdhtml.EscapeString(profileName))
	fmt.Fprintf(&sb, "<tr><td>Автор</td><td>%s</td></tr>\n", stdhtml.EscapeString(authorName))
	sb.WriteString("</table></details>")
	return sb.String()
}

func buildProfileCommentTelegramText(profileID, profileName, authorName, contentHTML string) string {
	metadataTable := buildProfileTelegramMetadataTable(resolveProfileURL(profileID), resolveProfileName(profileName), authorName)

	var sb strings.Builder
	sb.WriteString("<p>💬 Новый комментарий в профиле</p>\n")
	sb.WriteString(metadataTable)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "<details open><summary>Текст</summary>%s</details>", htmlToTelegramHTML(contentHTML))

	return truncateTelegramText(sb.String(), 30000)
}

func buildEditedProfileCommentTelegramText(profileID, profileName, authorName, oldContentHTML, newContentHTML string) string {
	metadataTable := buildProfileTelegramMetadataTable(resolveProfileURL(profileID), resolveProfileName(profileName), authorName)

	var sb strings.Builder
	sb.WriteString("<p>📝 Новая редакция комментария в профиле</p>\n")
	sb.WriteString(metadataTable)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "<details><summary>Старая версия</summary>%s</details>\n", htmlToTelegramHTML(oldContentHTML))
	fmt.Fprintf(&sb, "<details open><summary>Текст</summary>%s</details>", htmlToTelegramHTML(newContentHTML))

	return truncateTelegramText(sb.String(), 30000)
}
