package templates

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderSchemaNovel_Escaping(t *testing.T) {
	data := SchemaNovelData{
		Domain:      "https://kappalib.rip",
		Canonical:   "https://kappalib.rip/novel-1",
		Title:       `Novel "Special" Title`,
		Description: "Line 1\nLine 2\t\"Quotes\" </script><script>alert(1)</script>",
		Novel: SchemaNovel{
			ID:       "nvl_1",
			Title:    `Novel "Quotes" & More`,
			TitleEn:  `English "Title"`,
			Author:   `Author "Name"`,
			Status:   "ongoing",
			CoverURL: "https://s3.kappalib.rip/cover.jpg",
		},
	}

	rendered, err := RenderSchemaNovel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(rendered, "<script type=\"application/ld+json\">\n") {
		t.Errorf("missing script opening tag")
	}
	if !strings.HasSuffix(rendered, "\n</script>") {
		t.Errorf("missing script closing tag")
	}

	jsonStr := strings.TrimSuffix(strings.TrimPrefix(rendered, "<script type=\"application/ld+json\">\n"), "\n</script>")
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v, output: %s", err, jsonStr)
	}
}

func TestRenderSchemaChapter_Escaping(t *testing.T) {
	data := SchemaChapterData{
		Domain:       "https://kappalib.rip",
		Canonical:    "https://kappalib.rip/nvl_1/chapter/ch_1",
		Description:  "Chapter \"Desc\"\nNew line",
		ChapterTitle: "Глава 1: \"Начало\"",
		ChapterNum:   1,
		Novel: SchemaNovel{
			ID:    "nvl_1",
			Title: "Название \"Новеллы\"",
		},
	}

	rendered, err := RenderSchemaChapter(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonStr := strings.TrimSuffix(strings.TrimPrefix(rendered, "<script type=\"application/ld+json\">\n"), "\n</script>")
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v", err)
	}
}

func TestRenderSchemaWebsite_Escaping(t *testing.T) {
	data := SchemaWebsiteData{
		Domain:      "https://kappalib.rip",
		Canonical:   "https://kappalib.rip/catalog",
		Title:       "Каталог \"Новелл\"",
		Description: "Описание \"сайта\"\nС переходом",
	}

	rendered, err := RenderSchemaWebsite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonStr := strings.TrimSuffix(strings.TrimPrefix(rendered, "<script type=\"application/ld+json\">\n"), "\n</script>")
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v", err)
	}
}

func TestInitAndTextTemplates(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	robots, err := RenderRobots(RobotsData{Domain: "https://kappalib.rip"})
	if err != nil || !strings.Contains(robots, "https://kappalib.rip/sitemap.xml") {
		t.Errorf("RenderRobots failed or missing sitemap link: %v, %s", err, robots)
	}

	sitemap, err := RenderSitemap(SitemapData{
		Domain: "https://kappalib.rip",
		StaticPages: []StaticPage{
			{Path: "catalog"},
		},
	})
	if err != nil || !strings.Contains(sitemap, "<loc>https://kappalib.rip/catalog</loc>") {
		t.Errorf("RenderSitemap failed: %v, %s", err, sitemap)
	}

	key, err := RenderIndexNowKey(IndexNowKeyData{Key: "test-key-123"})
	if err != nil || strings.TrimSpace(key) != "test-key-123" {
		t.Errorf("RenderIndexNowKey failed: %v, %s", err, key)
	}
}
