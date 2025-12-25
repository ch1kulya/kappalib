package templates

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed *.tmpl
var FS embed.FS

var (
	robotsTmpl        *template.Template
	sitemapTmpl       *template.Template
	schemaWebsiteTmpl *template.Template
	schemaNovelTmpl   *template.Template
	schemaChapterTmpl *template.Template
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

	schemaWebsiteTmpl, err = template.ParseFS(FS, "schema_website.html.tmpl")
	if err != nil {
		return err
	}

	schemaNovelTmpl, err = template.ParseFS(FS, "schema_novel.html.tmpl")
	if err != nil {
		return err
	}

	schemaChapterTmpl, err = template.ParseFS(FS, "schema_chapter.html.tmpl")
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
	var buf bytes.Buffer
	if err := schemaWebsiteTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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
	var buf bytes.Buffer
	if err := schemaNovelTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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
	var buf bytes.Buffer
	if err := schemaChapterTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
