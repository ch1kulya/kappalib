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
