package views

import (
	"fmt"
	"html"
	"math"
	"math/rand/v2"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/models"
)

func MapStatus(status string) string {
	s := strings.ToLower(status)
	switch s {
	case "ongoing":
		return "Онгоинг"
	case "completed":
		return "Завершено"
	case "announced":
		return "Анонс"
	default:
		if len(status) > 0 {
			return strings.ToUpper(status[:1]) + status[1:]
		}
		return status
	}
}

func GetSortLabel(sort string) string {
	switch sort {
	case "newest":
		return "Новые"
	case "oldest":
		return "Старые"
	case "large":
		return "Большие"
	case "small":
		return "Маленькие"
	case "alphabet":
		return "Алфавитный порядок"
	case "created":
		return "Недавно добавленные"
	case "relevance":
		return "Релевантные"
	case "popular":
		return "Популярные"
	default:
		return "Популярные"
	}
}

func DerefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

const defaultCover = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='300'%3E%3Crect fill='%23ecf0f1' width='200' height='300'/%3E%3C/svg%3E"

func ResolveCover(coverURL *string) string {
	if coverURL == nil {
		return defaultCover
	}

	trimmed := strings.TrimSpace(*coverURL)
	if trimmed == "" {
		return defaultCover
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return defaultCover
	}

	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Host == "" {
		return defaultCover
	}

	return trimmed
}

func CalculatePagination(current, total int) []int {
	if total <= 1 {
		return nil
	}

	var pages []int

	if total <= 7 {
		for i := 1; i <= total; i++ {
			pages = append(pages, i)
		}
		return pages
	}

	pages = append(pages, 1)

	start := current - 2
	end := current + 2

	if start <= 2 {
		start = 2
		end = 5
	}

	if end >= total-1 {
		end = total - 1
		start = total - 4
	}

	if start > 2 {
		pages = append(pages, -1)
	}

	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	if end < total-1 {
		pages = append(pages, -1)
	}

	pages = append(pages, total)

	return pages
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func pluralize(n int, one, two, five string) string {
	n = int(math.Abs(float64(n))) % 100
	n1 := n % 10
	if n > 10 && n < 20 {
		return five
	}
	if n1 > 1 && n1 < 5 {
		return two
	}
	if n1 == 1 {
		return one
	}
	return five
}

func FormatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	diff := time.Since(t)
	seconds := int(diff.Seconds())
	minutes := int(diff.Minutes())
	hours := int(diff.Hours())
	days := int(hours / 24)

	switch {
	case seconds < 60:
		return "только что"
	case minutes < 60:
		return fmt.Sprintf("%d %s назад", minutes, pluralize(minutes, "минуту", "минуты", "минут"))
	case hours < 24:
		return fmt.Sprintf("%d %s назад", hours, pluralize(hours, "час", "часа", "часов"))
	case days < 30:
		return fmt.Sprintf("%d %s назад", days, pluralize(days, "день", "дня", "дней"))
	case days < 365:
		months := int(days / 30)
		return fmt.Sprintf("%d %s назад", months, pluralize(months, "месяц", "месяца", "месяцев"))
	default:
		years := int(days / 365)
		return fmt.Sprintf("%d %s назад", years, pluralize(years, "год", "года", "лет"))
	}
}

var FontOptions = []FontOption{
	{Value: "default", Label: "Стандартный", Family: "inherit"},
	{Value: "literata", Label: "Literata", Family: "Literata, serif"},
	{Value: "nunito", Label: "Nunito", Family: "Nunito, serif"},
	{Value: "merriweather", Label: "Merriweather", Family: "Merriweather, serif"},
	{Value: "lora", Label: "Lora", Family: "Lora, serif"},
	{Value: "pt-serif", Label: "PT Serif", Family: "PT Serif, serif"},
	{Value: "open-sans", Label: "Open Sans", Family: "Open Sans, sans-serif"},
	{Value: "roboto", Label: "Roboto", Family: "Roboto, sans-serif"},
}

var ColorSchemeOptions = []ColorSchemeOption{
	{Value: "default", Label: "Стандартная"},
	{Value: "catppuccin", Label: "Catppuccin"},
	{Value: "gruvbox", Label: "Gruvbox"},
	{Value: "nord", Label: "Nord"},
	{Value: "rosepine", Label: "Rosé Pine"},
	{Value: "tokyonight", Label: "Tokyo Night"},
	{Value: "everforest", Label: "Everforest"},
	{Value: "flexoki", Label: "Flexoki"},
	{Value: "sonokai", Label: "Sonokai"},
	{Value: "oxocarbon", Label: "Oxocarbon"},
}

var ListStatuses = []ListStatusOption{
	{Slug: "favorite", Label: "Избранное"},
	{Slug: "reading", Label: "Читаю"},
	{Slug: "rereading", Label: "Перечитываю"},
	{Slug: "planned", Label: "Запланировано"},
	{Slug: "on_hold", Label: "Отложено"},
	{Slug: "completed", Label: "Прочитано"},
	{Slug: "dropped", Label: "Брошено"},
}

var FontURLs = map[string]string{
	"literata":     "https://cdn.jsdelivr.net/npm/@fontsource/literata@5/index.min.css",
	"nunito":       "https://cdn.jsdelivr.net/npm/@fontsource/nunito@5/index.min.css",
	"merriweather": "https://cdn.jsdelivr.net/npm/@fontsource/merriweather@5/index.min.css",
	"lora":         "https://cdn.jsdelivr.net/npm/@fontsource/lora@5/index.min.css",
	"pt-serif":     "https://cdn.jsdelivr.net/npm/@fontsource/pt-serif@5/index.min.css",
	"open-sans":    "https://cdn.jsdelivr.net/npm/@fontsource/open-sans@5/index.min.css",
	"roboto":       "https://cdn.jsdelivr.net/npm/@fontsource/roboto@5/index.min.css",
}

var DefaultReaderSettings = ReaderSettings{
	Theme:        "auto",
	ColorScheme:  "default",
	FontSize:     18,
	FontFamily:   "default",
	Indent:       0,
	Density:      "normal",
	Justify:      false,
	ShowComments: true,
}

func GetFontFamily(fontKey string) string {
	for _, f := range FontOptions {
		if f.Value == fontKey {
			return f.Family
		}
	}
	return "inherit"
}

func GetFontLabel(fontKey string) string {
	for _, f := range FontOptions {
		if f.Value == fontKey {
			return f.Label
		}
	}
	return "Стандартный"
}

func GetFontURL(fontKey string) string {
	return FontURLs[fontKey]
}

func IsValidColorScheme(scheme string) bool {
	for _, s := range ColorSchemeOptions {
		if s.Value == scheme {
			return true
		}
	}
	return false
}

func BrandIconURL(scheme, file string) string {
	if !IsValidColorScheme(scheme) {
		scheme = "default"
	}
	return "/assets/icons/" + scheme + "/" + file
}

func chapterBackURL(novel *models.Novel) string {
	if novel == nil {
		return "/downloads"
	}
	return "/" + novel.ID
}

func chapterContentClasses(settings ReaderSettings) string {
	classes := "chapter-content"
	classes += " density-" + settings.Density
	if settings.Justify {
		classes += " justify-text"
	}
	return classes
}

func chapterContentStyle(settings ReaderSettings) string {
	style := fmt.Sprintf("font-size: %.4frem;", float64(settings.FontSize)/16)
	if settings.FontFamily != "default" {
		style += fmt.Sprintf(" font-family: %s;", GetFontFamily(settings.FontFamily))
	}
	if settings.Indent > 0 {
		style += fmt.Sprintf(" --reader-indent: %dem;", settings.Indent)
	} else {
		style += " --reader-indent: 0;"
	}
	return style
}

func chapterTitleStyle(settings ReaderSettings) string {
	baseFontSize := float64(settings.FontSize)
	titleRatio := 1.5 / 1.125
	titleFontSize := baseFontSize * titleRatio

	baseMarginRem := 2.0
	marginRatio := baseFontSize / 18
	titleMargin := baseMarginRem * marginRatio

	style := fmt.Sprintf("font-size: %.4frem; margin-bottom: %.4frem;", titleFontSize/16, titleMargin)
	if settings.FontFamily != "default" {
		style += fmt.Sprintf(" font-family: %s;", GetFontFamily(settings.FontFamily))
	}
	return style
}

func chapterTitleClasses(settings ReaderSettings) string {
	if settings.Justify {
		return "justify-text"
	}
	return ""
}

var volumeTagRe = regexp.MustCompile(`\[((?:Начало|Конец)(?:\s+\d+\s+тома)?)\]`)

func listStatusLabel(status string) string {
	for _, opt := range ListStatuses {
		if opt.Slug == status {
			return opt.Label
		}
	}
	return "В список"
}

func lsRemoveWrapStyle(status string) string {
	if status == "" {
		return "display: none;"
	}
	return "display: block;"
}

func FormatTitle(title string) string {
	if !strings.Contains(title, "[") {
		return html.EscapeString(title)
	}
	return volumeTagRe.ReplaceAllString(html.EscapeString(title), `<span class="volume-tag">[$1]</span>`)
}

func FormatChapterHeading(num int, title string) string {
	prefix := fmt.Sprintf("Глава %d", num)
	if title == "Без названия" {
		return prefix
	}
	return prefix + ": " + FormatTitle(title)
}

func FormatUpdateActionHTML(min, max, count int) string {
	if count == 1 {
		return fmt.Sprintf(`Добавлена <span class="update-log-num">%d</span> глава`, min)
	}
	return fmt.Sprintf(`Добавлены главы <span class="update-log-num">%d-%d</span>`, min, max)
}

func FormatNovelAdditionChaptersHTML(min, max, count int) string {
	if count == 1 || min == max {
		return fmt.Sprintf(`Новая новелла, добавлена <span class="update-log-num">%d</span> глава`, min)
	}
	return fmt.Sprintf(`Новая новелла, добавлены главы <span class="update-log-num">%d-%d</span>`, min, max)
}

func FormatGroupedUpdatesHTML(chapterCount, novelCount int) string {
	action := "Добавлено"
	if chapterCount%10 == 1 && chapterCount%100 != 11 {
		action = "Добавлена"
	}
	chWord := pluralize(chapterCount, "глава", "главы", "глав")
	novWord := pluralize(novelCount, "новеллы", "новелл", "новелл")
	return fmt.Sprintf(`%s <span class="update-log-num">%d</span> %s для <span class="update-log-num">%d</span> %s`,
		action, chapterCount, chWord, novelCount, novWord)
}

func FormatUpdateBody(body string) []string {
	const tag = "!update:"
	idx := strings.Index(body, tag)
	if idx == -1 {
		return nil
	}
	start := idx + len(tag)
	rest := body[start:]
	before, _, ok := strings.Cut(rest, "\n")
	var content string
	if !ok {
		content = rest
	} else {
		content = before
	}
	content = strings.TrimSpace(content)

	parts := strings.Split(content, ";")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func pluralizeNovels(n int) string {
	return pluralize(n, "новелла", "новеллы", "новелл")
}

func buildWarningList(hasSelfHarm, hasDrugUsage, hasSexualViolence, hasGraphicSex, hasProfanity bool) string {
	var warnings []string
	if hasSelfHarm {
		warnings = append(warnings, "суицидальные темы", "селфхарм")
	}
	if hasDrugUsage {
		warnings = append(warnings, "употребление наркотиков")
	}
	if hasSexualViolence {
		warnings = append(warnings, "сексуальное насилие")
	}
	if hasGraphicSex {
		warnings = append(warnings, "графичную эротику")
	}
	if hasProfanity {
		warnings = append(warnings, "нецензурную брань")
	}
	if len(warnings) == 0 {
		return ""
	}
	if len(warnings) == 1 {
		return warnings[0] + "."
	}
	return strings.Join(warnings[:len(warnings)-1], ", ") + " и " + warnings[len(warnings)-1] + "."
}

func shouldShowAnnouncement() bool {
	return rand.IntN(7) == 0
}

func isExternalURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
}

var heatmapLevelThresholds = []int{1, 10 * 60, 30 * 60, 90 * 60}

var monthShortNames = [...]string{"Янв", "Фев", "Мар", "Апр", "Май", "Июн", "Июл", "Авг", "Сен", "Окт", "Ноя", "Дек"}

var monthGenitiveNames = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}

func heatmapLevel(seconds int) int {
	level := 0
	for i, threshold := range heatmapLevelThresholds {
		if seconds >= threshold {
			level = i + 1
		}
	}
	return level
}

func BuildActivityHeatmap(days []models.ActivityDay) ActivityHeatmap {
	var heatmap ActivityHeatmap
	for _, day := range days {
		weekday := (int(day.Date.Weekday()) + 6) % 7
		if len(heatmap.Weeks) == 0 {
			week := make([]HeatmapCell, weekday, 7)
			for i := range week {
				week[i].Empty = true
			}
			heatmap.Weeks = append(heatmap.Weeks, week)
		} else if weekday == 0 {
			heatmap.Weeks = append(heatmap.Weeks, make([]HeatmapCell, 0, 7))
		}
		last := len(heatmap.Weeks) - 1
		heatmap.Weeks[last] = append(heatmap.Weeks[last], HeatmapCell{
			Date:    day.Date,
			Seconds: day.Seconds,
			Level:   heatmapLevel(day.Seconds),
		})
		heatmap.TotalSeconds += day.Seconds
	}
	heatmap.Months = buildHeatmapMonths(heatmap.Weeks)
	return heatmap
}

func buildHeatmapMonths(weeks [][]HeatmapCell) []HeatmapMonth {
	months := make([]HeatmapMonth, 0, 13)
	prevMonth := time.Month(0)
	for col, week := range weeks {
		for _, cell := range week {
			if cell.Empty {
				continue
			}
			if cell.Date.Month() != prevMonth {
				prevMonth = cell.Date.Month()
				months = append(months, HeatmapMonth{Label: monthShortNames[prevMonth-1], Column: col})
			}
			break
		}
	}
	if len(months) > 1 && months[1].Column-months[0].Column < 2 {
		months = months[1:]
	}
	if n := len(months); n > 0 && months[n-1].Column > len(weeks)-2 {
		months = months[:n-1]
	}
	return months
}

func heatmapBodyStyle(weeks int) string {
	return fmt.Sprintf("--weeks: %d", max(weeks, 1))
}

func heatmapColumnStyle(column int) string {
	return fmt.Sprintf("--col: %d", column)
}

func heatmapCellLabel(cell HeatmapCell) string {
	if cell.Seconds == 0 {
		return "Нет активности · " + FormatDate(cell.Date)
	}
	return FormatDuration(cell.Seconds) + " · " + FormatDate(cell.Date)
}

func heatmapSummary(totalSeconds int) string {
	if totalSeconds == 0 {
		return "Нет активности за последний год"
	}
	return FormatDuration(totalSeconds) + " за последний год"
}

func FormatDuration(seconds int) string {
	if seconds <= 0 {
		return "0 мин"
	}
	if seconds < 60 {
		return "меньше минуты"
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	switch {
	case hours == 0:
		return fmt.Sprintf("%d мин", minutes)
	case minutes == 0:
		return fmt.Sprintf("%d ч", hours)
	default:
		return fmt.Sprintf("%d ч %d мин", hours, minutes)
	}
}

func FormatDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), monthGenitiveNames[t.Month()-1], t.Year())
}

func AvatarURL(userID string, hasCustomAvatar bool, avatarSeed string, avatarUpdatedAt int64) string {
	if hasCustomAvatar {
		return fmt.Sprintf("%s/avatars/%s.jpg?v=%d", os.Getenv("S3_PUBLIC_URL"), userID, avatarUpdatedAt)
	}
	return "https://api.dicebear.com/9.x/bottts-neutral/svg?seed=" + url.QueryEscape(avatarSeed) + "&backgroundType=solid,gradientLinear"
}

func streakLabel(days int) string {
	return fmt.Sprintf("%d %s подряд", days, pluralize(days, "день", "дня", "дней"))
}

func achievementTierLabel(tier int) string {
	return fmt.Sprintf("x%d", tier)
}

var compactNumberUnits = []struct {
	value  int64
	suffix string
}{
	{1_000_000_000, "млрд"},
	{1_000_000, "млн"},
	{1_000, "тыс."},
}

func FormatCompactNumber(n int64) string {
	for _, unit := range compactNumberUnits {
		if n < unit.value {
			continue
		}
		if n >= 10*unit.value {
			return fmt.Sprintf("%d %s", n/unit.value, unit.suffix)
		}
		tenths := n * 10 / unit.value
		if tenths%10 == 0 {
			return fmt.Sprintf("%d %s", tenths/10, unit.suffix)
		}
		return fmt.Sprintf("%d,%d %s", tenths/10, tenths%10, unit.suffix)
	}
	return strconv.FormatInt(n, 10)
}

func groupDigits(n int64) string {
	digits := strconv.FormatInt(n, 10)
	sign := ""
	if n < 0 {
		sign, digits = "-", digits[1:]
	}
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteRune('\u00a0')
		}
		b.WriteRune(d)
	}
	return sign + b.String()
}

func exactCountLabel(n int64, one, two, five string) string {
	return fmt.Sprintf("%s %s", groupDigits(n), pluralize(int(n%100), one, two, five))
}

func compactCountWord(n int64, one, two, five string) string {
	if n >= 1_000 {
		return five
	}
	return pluralize(int(n), one, two, five)
}

func viewsLabel(views int64) string {
	return exactCountLabel(views, "просмотр", "просмотра", "просмотров")
}

func viewsWord(views int64) string {
	return compactCountWord(views, "просмотр", "просмотра", "просмотров")
}

type catalogStatItem struct {
	Value          int64
	One, Two, Five string
}

func catalogStatItems(stats models.CatalogStats) []catalogStatItem {
	return []catalogStatItem{
		{stats.NovelsCount, "новелла", "новеллы", "новелл"},
		{stats.ChaptersCount, "глава", "главы", "глав"},
		{stats.SourcesCount, "источник", "источника", "источников"},
		{stats.CharactersCount, "знак", "знака", "знаков"},
	}
}

func catalogRangeBadge(from, to *int) string {
	switch {
	case from != nil && to != nil:
		return fmt.Sprintf("%d–%d", *from, *to)
	case from != nil:
		return fmt.Sprintf("от %d", *from)
	case to != nil:
		return fmt.Sprintf("до %d", *to)
	default:
		return ""
	}
}

func catalogCountBadge(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func optionalIntValue(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}
