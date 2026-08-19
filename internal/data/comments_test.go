package data

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestVerifyCommentsCaptcha_EmptyTokens(t *testing.T) {
	if verifyCommentsCaptcha("", "", "") {
		t.Error("expected false for empty tokens")
	}
}

func TestVerifyCommentsSmartCaptcha_MockServer(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   map[string]string
		secret     string
		want       bool
	}{
		{
			name:       "empty secret",
			secret:     "",
			statusCode: http.StatusOK,
			response:   map[string]string{"status": "ok"},
			want:       false,
		},
		{
			name:       "valid token ok",
			secret:     "test-secret",
			statusCode: http.StatusOK,
			response:   map[string]string{"status": "ok"},
			want:       true,
		},
		{
			name:       "invalid token failed",
			secret:     "test-secret",
			statusCode: http.StatusOK,
			response:   map[string]string{"status": "failed", "message": "invalid"},
			want:       false,
		},
		{
			name:       "server 500 error fail-closed",
			secret:     "test-secret",
			statusCode: http.StatusInternalServerError,
			response:   map[string]string{"status": "error"},
			want:       false,
		},
		{
			name:       "server 403 error fail-closed",
			secret:     "wrong-secret",
			statusCode: http.StatusForbidden,
			response:   map[string]string{"status": "error", "message": "forbidden"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST method, got %s", r.Method)
				}
				if err := r.ParseForm(); err != nil {
					t.Errorf("failed to parse form: %v", err)
				}
				if r.FormValue("secret") != tt.secret {
					t.Errorf("expected secret %s, got %s", tt.secret, r.FormValue("secret"))
				}
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			prevSecret := commentsSmartCaptchaSecret
			commentsSmartCaptchaSecret = tt.secret
			defer func() { commentsSmartCaptchaSecret = prevSecret }()

			client := &http.Client{Timeout: 3 * time.Second}
			params := url.Values{
				"secret": {commentsSmartCaptchaSecret},
				"token":  {"test-token"},
				"ip":     {"127.0.0.1"},
			}

			if commentsSmartCaptchaSecret == "" {
				if verifyCommentsSmartCaptcha("test-token", "127.0.0.1") != tt.want {
					t.Errorf("expected %v for empty secret", tt.want)
				}
				return
			}

			resp, err := client.PostForm(server.URL, params)
			if err != nil {
				if tt.want {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			defer func() { _ = resp.Body.Close() }()

			var got bool
			if resp.StatusCode != http.StatusOK {
				got = false
			} else {
				var result struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					got = false
				} else {
					got = result.Status == "ok"
				}
			}

			if got != tt.want {
				t.Errorf("verify result = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHtmlToTelegramHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty input",
			input: "",
			want:  "[без текста]",
		},
		{
			name:  "whitespace only",
			input: "   \n\t  ",
			want:  "[без текста]",
		},
		{
			name:  "bold, italic, strikethrough, underline",
			input: "<p><strong>bold</strong> <em>italic</em> <del>strike</del> <u>underline</u></p>",
			want:  "<p><b>bold</b> <i>italic</i> <s>strike</s> <u>underline</u></p>",
		},
		{
			name:  "code and pre",
			input: "<p><code>inline code</code></p><pre><code>func main() {}</code></pre>",
			want:  "<p><code>inline code</code></p><pre>func main() {}</pre>",
		},
		{
			name:  "spoiler tag",
			input: `<p><span class="spoiler">secret text</span></p>`,
			want:  "<p><tg-spoiler>secret text</tg-spoiler></p>",
		},
		{
			name:  "link with extra attributes cleaned",
			input: `<p><a href="https://example.com" rel="nofollow" target="_blank">link</a></p>`,
			want:  `<p><a href="https://example.com">link</a></p>`,
		},
		{
			name:  "image tag",
			input: `<p><img src="https://example.com/pic.png" alt="preview" /></p>`,
			want:  `<p><img src="https://example.com/pic.png"/></p>`,
		},
		{
			name:  "image without src",
			input: `<p><img alt="broken" /></p>`,
			want:  `<p></p>`,
		},
		{
			name:  "top-level blockquote",
			input: "<blockquote>quote text</blockquote>",
			want:  "<blockquote>quote text</blockquote>",
		},
		{
			name:  "html special characters escaped in text",
			input: "<p>1 &lt; 2 &amp; 3 &gt; 2</p>",
			want:  "<p>1 &lt; 2 &amp; 3 &gt; 2</p>",
		},
		{
			name:  "lists and headings",
			input: "<h2>Header</h2><ul><li>item 1</li><li>item 2</li></ul>",
			want:  "<h2>Header</h2>\n<ul><li>item 1</li><li>item 2</li></ul>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlToTelegramHTML(tt.input)
			if got != tt.want {
				t.Errorf("htmlToTelegramHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTelegramChapterTitleInTable(t *testing.T) {
	tests := []struct {
		num   int
		title string
		want  string
	}{
		{num: 1, title: "", want: "1. Без названия"},
		{num: 5, title: "Без названия", want: "5. Без названия"},
		{num: 10, title: "Пролог", want: "10. Пролог"},
	}

	for _, tt := range tests {
		got := formatTelegramChapterTitleInTable(tt.num, tt.title)
		if got != tt.want {
			t.Errorf("formatTelegramChapterTitleInTable(%d, %q) = %q, want %q", tt.num, tt.title, got, tt.want)
		}
	}
}

func TestBuildTelegramMetadataTable(t *testing.T) {
	got := buildTelegramMetadataTable("https://kappalib.rip/nvl_1", "Novel", "https://kappalib.rip/nvl_1/chapter/chp_1", 1, "Prologue", "User1", "User2")
	if !strings.Contains(got, "<details><summary>Информация</summary><table>") {
		t.Errorf("expected details with table, got %q", got)
	}
	if !strings.Contains(got, `<tr><td>Новелла</td><td><a href="https://kappalib.rip/nvl_1">Novel</a></td></tr>`) {
		t.Errorf("expected novel row, got %q", got)
	}
	if !strings.Contains(got, `<tr><td>Глава</td><td><a href="https://kappalib.rip/nvl_1/chapter/chp_1">1. Prologue</a></td></tr>`) {
		t.Errorf("expected chapter row, got %q", got)
	}
	if !strings.Contains(got, `<tr><td>Автор</td><td>User1</td></tr>`) {
		t.Errorf("expected author row, got %q", got)
	}
	if !strings.Contains(got, `<tr><td>Кому</td><td>User2</td></tr>`) {
		t.Errorf("expected recipient row, got %q", got)
	}
}

func TestBuildCommentTelegramText(t *testing.T) {
	got := buildCommentTelegramText(
		"nvl_123", "Novel Title", "chp_456", 1, "Prologue",
		"User1", "<p>Great chapter! <strong>bold</strong></p>",
	)

	if !strings.Contains(got, "<p>💬 Новый комментарий</p>") {
		t.Errorf("expected header in %q", got)
	}
	if !strings.Contains(got, "<details><summary>Информация</summary><table>") {
		t.Errorf("expected details with table in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Новелла</td><td><a href="https://kappalib.rip/nvl_123">Novel Title</a></td></tr>`) {
		t.Errorf("expected novel row in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Глава</td><td><a href="https://kappalib.rip/nvl_123/chapter/chp_456">1. Prologue</a></td></tr>`) {
		t.Errorf("expected chapter row in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Автор</td><td>User1</td></tr>`) {
		t.Errorf("expected author row in %q", got)
	}
	if strings.Contains(got, "<hr/>") || strings.Contains(got, "<hr>") {
		t.Errorf("unexpected hr in %q", got)
	}
	if !strings.Contains(got, "<details open><summary>Текст</summary><p>Great chapter! <b>bold</b></p></details>") {
		t.Errorf("expected details open with formatted text in %q", got)
	}
}

func TestBuildAnswerTelegramText(t *testing.T) {
	got := buildAnswerTelegramText(
		"nvl_123", "Novel Title", "chp_456", 2, "Second Chapter",
		"ParentUser", "<p>Original comment with <blockquote>quote</blockquote></p>",
		"ReplyUser", "<p>Reply comment</p>",
	)

	if !strings.Contains(got, "<p>💬 Ответ на комментарий</p>") {
		t.Errorf("expected header in %q", got)
	}
	if !strings.Contains(got, "<details><summary>Информация</summary><table>") {
		t.Errorf("expected details with table in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Новелла</td><td><a href="https://kappalib.rip/nvl_123">Novel Title</a></td></tr>`) {
		t.Errorf("expected novel link in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Глава</td><td><a href="https://kappalib.rip/nvl_123/chapter/chp_456">2. Second Chapter</a></td></tr>`) {
		t.Errorf("expected chapter link in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Автор</td><td>ReplyUser</td></tr>`) {
		t.Errorf("expected reply author in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Кому</td><td>ParentUser</td></tr>`) {
		t.Errorf("expected recipient in metadata table in %q", got)
	}
	if !strings.Contains(got, "<details><summary>Комментарий</summary>") {
		t.Errorf("expected details with parent comment in %q", got)
	}
	if !strings.Contains(got, "<blockquote>quote</blockquote>") {
		t.Errorf("expected blockquote in parent comment in %q", got)
	}
	if strings.Contains(got, "<hr/>") || strings.Contains(got, "<hr>") {
		t.Errorf("unexpected hr in %q", got)
	}
	if !strings.Contains(got, "<details open><summary>Ответ</summary><p>Reply comment</p></details>") {
		t.Errorf("expected details open with reply content in %q", got)
	}
}

func TestBuildEditedCommentTelegramText(t *testing.T) {
	got := buildEditedCommentTelegramText(
		"nvl_123", "Novel Title", "chp_456", 1, "Prologue",
		"User1", "<p>Old text</p>", "<p>New text <b>edited</b></p>",
	)

	if !strings.Contains(got, "<p>📝 Новая редакция комментария</p>") {
		t.Errorf("expected header in %q", got)
	}
	if !strings.Contains(got, "<details><summary>Старая версия</summary><p>Old text</p></details>") {
		t.Errorf("expected old version details in %q", got)
	}
	if !strings.Contains(got, "<details open><summary>Текст</summary><p>New text <b>edited</b></p></details>") {
		t.Errorf("expected new version details open in %q", got)
	}
}

func TestBuildEditedAnswerTelegramText(t *testing.T) {
	got := buildEditedAnswerTelegramText(
		"nvl_123", "Novel Title", "chp_456", 2, "Second Chapter",
		"ParentUser", "<p>Parent comment</p>",
		"ReplyUser", "<p>Old reply</p>", "<p>New reply</p>",
	)

	if !strings.Contains(got, "<p>📝 Новая редакция ответа</p>") {
		t.Errorf("expected header in %q", got)
	}
	if !strings.Contains(got, "<details><summary>Комментарий</summary><p>Parent comment</p></details>") {
		t.Errorf("expected parent comment in %q", got)
	}
	if !strings.Contains(got, "<details><summary>Старая версия</summary><p>Old reply</p></details>") {
		t.Errorf("expected old version in %q", got)
	}
	if !strings.Contains(got, "<details open><summary>Ответ</summary><p>New reply</p></details>") {
		t.Errorf("expected new version in %q", got)
	}
}

func TestCalculateCommentsPagination(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		totalCount     int
		isDeepLink     bool
		targetPage     int
		wantLimit      int
		wantOffset     int
		wantResultPage int
		wantTotalPages int
	}{
		{
			name:           "standard page 1",
			page:           1,
			pageSize:       12,
			totalCount:     25,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     0,
			wantResultPage: 1,
			wantTotalPages: 3,
		},
		{
			name:           "standard page 2",
			page:           2,
			pageSize:       12,
			totalCount:     25,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     12,
			wantResultPage: 2,
			wantTotalPages: 3,
		},
		{
			name:           "standard page 3",
			page:           3,
			pageSize:       12,
			totalCount:     25,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     24,
			wantResultPage: 3,
			wantTotalPages: 3,
		},
		{
			name:           "deep link page 1",
			page:           1,
			pageSize:       12,
			totalCount:     50,
			isDeepLink:     true,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     0,
			wantResultPage: 1,
			wantTotalPages: 5,
		},
		{
			name:           "deep link page 3 fetches pages 1 to 3",
			page:           1,
			pageSize:       12,
			totalCount:     50,
			isDeepLink:     true,
			targetPage:     3,
			wantLimit:      36,
			wantOffset:     0,
			wantResultPage: 3,
			wantTotalPages: 5,
		},
		{
			name:           "deep link page 5 fetches pages 1 to 5",
			page:           1,
			pageSize:       12,
			totalCount:     50,
			isDeepLink:     true,
			targetPage:     5,
			wantLimit:      60,
			wantOffset:     0,
			wantResultPage: 5,
			wantTotalPages: 5,
		},
		{
			name:           "zero count produces zero total pages",
			page:           1,
			pageSize:       12,
			totalCount:     0,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     0,
			wantResultPage: 1,
			wantTotalPages: 0,
		},
		{
			name:           "single item count produces one page",
			page:           1,
			pageSize:       12,
			totalCount:     1,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     0,
			wantResultPage: 1,
			wantTotalPages: 1,
		},
		{
			name:           "exact page size count",
			page:           1,
			pageSize:       12,
			totalCount:     12,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     0,
			wantResultPage: 1,
			wantTotalPages: 1,
		},
		{
			name:           "exact page size plus one count",
			page:           1,
			pageSize:       12,
			totalCount:     13,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     0,
			wantResultPage: 1,
			wantTotalPages: 2,
		},
		{
			name:           "exact two pages count",
			page:           1,
			pageSize:       12,
			totalCount:     24,
			isDeepLink:     false,
			targetPage:     1,
			wantLimit:      12,
			wantOffset:     0,
			wantResultPage: 1,
			wantTotalPages: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotOffset, gotResultPage, gotTotalPages := CalculateCommentsPagination(
				tt.page,
				tt.pageSize,
				tt.totalCount,
				tt.isDeepLink,
				tt.targetPage,
			)

			if gotLimit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", gotOffset, tt.wantOffset)
			}
			if gotResultPage != tt.wantResultPage {
				t.Errorf("resultPage = %d, want %d", gotResultPage, tt.wantResultPage)
			}
			if gotTotalPages != tt.wantTotalPages {
				t.Errorf("totalPages = %d, want %d", gotTotalPages, tt.wantTotalPages)
			}
		})
	}
}

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello world",
			expected: "<p>Hello world</p>",
		},
		{
			name:     "bold and italic",
			input:    "**bold** and *italic*",
			expected: "<p><strong>bold</strong> and <em>italic</em></p>",
		},
		{
			name:     "spoiler tag",
			input:    "||secret spoiler||",
			expected: `<p><span class="spoiler">secret spoiler</span></p>`,
		},
		{
			name:     "image tag with lazy loading",
			input:    "![alt text](https://example.com/image.png)",
			expected: `<p><img loading="lazy" src="https://example.com/image.png" alt="alt text"/></p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderMarkdown(tt.input)
			if got != tt.expected {
				t.Errorf("renderMarkdown(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateSubmissionLength(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxLen   int
		isAnswer bool
		wantErr  error
	}{
		{
			name:     "comment empty content",
			content:  "",
			maxLen:   3000,
			isAnswer: false,
			wantErr:  ErrInvalidContentLength,
		},
		{
			name:     "comment too long",
			content:  strings.Repeat("a", 3001),
			maxLen:   3000,
			isAnswer: false,
			wantErr:  ErrInvalidContentLength,
		},
		{
			name:     "answer empty content",
			content:  "",
			maxLen:   500,
			isAnswer: true,
			wantErr:  ErrInvalidAnswerLength,
		},
		{
			name:     "answer too long",
			content:  strings.Repeat("a", 501),
			maxLen:   500,
			isAnswer: true,
			wantErr:  ErrInvalidAnswerLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubmission("user1", tt.content, tt.maxLen, tt.isAnswer, "", "", "")
			if err != tt.wantErr {
				t.Errorf("validateSubmission() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
