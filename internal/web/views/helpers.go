package views

import (
	"fmt"
	"math"
	"strings"
	"time"
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
		return "Сначала новые"
	case "large":
		return "Сначала большие"
	case "small":
		return "Сначала маленькие"
	case "alphabet":
		return "По алфавиту"
	case "created":
		return "Недавно добавленные"
	default:
		return "Сначала старые"
	}
}

func DerefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ResolveCover(url *string) string {
	if url != nil && *url != "" {
		return *url
	}
	return "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='300'%3E%3Crect fill='%23ecf0f1' width='200' height='300'/%3E%3C/svg%3E"
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
	Theme:      "auto",
	FontSize:   18,
	FontFamily: "default",
	Indent:     0,
	Density:    "normal",
	Justify:    false,
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
