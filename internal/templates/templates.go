package templates

import (
	"bytes"
	"embed"
	"encoding/json"
	"text/template"
)

//go:embed *.tmpl
var FS embed.FS

var (
	robotsTmpl      *template.Template
	sitemapTmpl     *template.Template
	indexNowKeyTmpl *template.Template
)

func Init() error {
	var err error

	robotsTmpl, err = template.ParseFS(FS, "robots.txt.tmpl")
	if err != nil {
		return err
	}

	sitemapTmpl, err = template.ParseFS(FS, "sitemap.xml.tmpl")
	if err != nil {
		return err
	}

	indexNowKeyTmpl, err = template.ParseFS(FS, "indexnow.txt.tmpl")
	if err != nil {
		return err
	}

	return nil
}

type RobotsData struct {
	Domain string
}

func RenderRobots(data RobotsData) (string, error) {
	var buf bytes.Buffer
	if err := robotsTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type StaticPage struct {
	Path string
}

type SitemapNovel struct {
	ID        string
	CreatedAt interface{ Format(string) string }
}

type SitemapData struct {
	Domain      string
	StaticPages []StaticPage
	Novels      []SitemapNovel
}

func RenderSitemap(data SitemapData) (string, error) {
	var buf bytes.Buffer
	if err := sitemapTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type SchemaWebsiteData struct {
	Domain      string
	Canonical   string
	Title       string
	Description string
}

func RenderSchemaWebsite(data SchemaWebsiteData) (string, error) {
	graph := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{
				"@type":       "WebSite",
				"url":         data.Domain,
				"name":        "kappalib",
				"description": data.Description,
				"inLanguage":  "ru-RU",
			},
			map[string]any{
				"@type": "Organization",
				"name":  "kappalib",
				"url":   data.Domain,
				"logo": map[string]any{
					"@type": "ImageObject",
					"url":   "https://s3.kappalib.rip/favicon.ico",
				},
				"contactPoint": map[string]any{
					"@type":       "ContactPoint",
					"email":       "support@kappalib.rip",
					"contactType": "customer service",
				},
			},
			map[string]any{
				"@type":       "CollectionPage",
				"@id":         data.Canonical,
				"url":         data.Canonical,
				"name":        data.Title,
				"description": data.Description,
				"inLanguage":  "ru-RU",
			},
		},
	}
	b, err := json.Marshal(graph)
	if err != nil {
		return "", err
	}
	return "<script type=\"application/ld+json\">\n" + string(b) + "\n</script>", nil
}

type SchemaNovelData struct {
	Domain      string
	Canonical   string
	Title       string
	Description string
	Novel       SchemaNovel
}

type SchemaNovel struct {
	ID       string
	Title    string
	TitleEn  string
	Author   string
	Status   string
	CoverURL string
}

func RenderSchemaNovel(data SchemaNovelData) (string, error) {
	book := map[string]any{
		"@type":               "Book",
		"url":                 data.Canonical,
		"name":                data.Novel.Title,
		"description":         data.Description,
		"inLanguage":          "ru-RU",
		"isAccessibleForFree": true,
	}
	if data.Novel.TitleEn != "" {
		book["alternateName"] = data.Novel.TitleEn
	}
	if data.Novel.CoverURL != "" {
		book["image"] = data.Novel.CoverURL
	}
	if data.Novel.Author != "" {
		book["author"] = map[string]any{
			"@type": "Person",
			"name":  data.Novel.Author,
		}
	}
	if data.Novel.Status != "" {
		book["workExample"] = map[string]any{
			"@type":       "Book",
			"bookEdition": "Веб-новелла",
			"bookFormat":  "https://schema.org/EBook",
		}
	}

	graph := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{
				"@type": "BreadcrumbList",
				"itemListElement": []any{
					map[string]any{
						"@type":    "ListItem",
						"position": 1,
						"name":     "Главная",
						"item":     data.Domain,
					},
					map[string]any{
						"@type":    "ListItem",
						"position": 2,
						"name":     data.Novel.Title,
						"item":     data.Canonical,
					},
				},
			},
			book,
		},
	}
	b, err := json.Marshal(graph)
	if err != nil {
		return "", err
	}
	return "<script type=\"application/ld+json\">\n" + string(b) + "\n</script>", nil
}

type SchemaChapterData struct {
	Domain       string
	Canonical    string
	Description  string
	ChapterTitle string
	ChapterNum   int
	Novel        SchemaNovel
}

func RenderSchemaChapter(data SchemaChapterData) (string, error) {
	graph := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{
				"@type": "BreadcrumbList",
				"itemListElement": []any{
					map[string]any{
						"@type":    "ListItem",
						"position": 1,
						"name":     "Главная",
						"item":     data.Domain,
					},
					map[string]any{
						"@type":    "ListItem",
						"position": 2,
						"name":     data.Novel.Title,
						"item":     data.Domain + "/" + data.Novel.ID,
					},
					map[string]any{
						"@type":    "ListItem",
						"position": 3,
						"name":     data.ChapterTitle,
						"item":     data.Canonical,
					},
				},
			},
			map[string]any{
				"@type":       "Chapter",
				"url":         data.Canonical,
				"name":        data.ChapterTitle,
				"description": data.Description,
				"position":    data.ChapterNum,
				"isPartOf": map[string]any{
					"@type": "Book",
					"name":  data.Novel.Title,
				},
				"inLanguage":          "ru-RU",
				"isAccessibleForFree": true,
			},
		},
	}
	b, err := json.Marshal(graph)
	if err != nil {
		return "", err
	}
	return "<script type=\"application/ld+json\">\n" + string(b) + "\n</script>", nil
}

type IndexNowKeyData struct {
	Key string
}

func RenderIndexNowKey(data IndexNowKeyData) (string, error) {
	var buf bytes.Buffer
	if err := indexNowKeyTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
