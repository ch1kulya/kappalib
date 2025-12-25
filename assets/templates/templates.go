package templates

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed *.tmpl
var FS embed.FS

var (
	robotsTmpl  *template.Template
	sitemapTmpl *template.Template
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
