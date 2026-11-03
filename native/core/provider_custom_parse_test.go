package core

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// 合成 fixture：不放入真实论坛页面（含用户个人发帖内容，禁止入库）
const fixtureSpacePage = `<html><body>
<a href="forum.php?mod=viewthread&tid=1001">【视频】第一帖标题</a>
<a href="forum.php?mod=viewthread&tid=1002">【视频】第二帖标题</a>
<a href="forum.php?mod=viewthread&tid=1001">重复链接应去重</a>
<a href="forum.php?mod=space&uid=1">空间链接不算目录</a>
<a href="home.php?mod=space&uid=1&do=thread&page=2">下一页</a>
</body></html>`

const fixtureThreadPage = `<html><head><title>【视频】第一帖标题 - 测试论坛</title></head><body>
正文 <a href="https://objectstorage.us-phoenix-1.oraclecloud.com/n/ns/b/opvideo/o/316/%E8%A7%86%E9%A2%916-a.MP4.min.mp4">视频6-a</a>
又一段 <a href="https://objectstorage.us-phoenix-1.oraclecloud.com/n/ns/b/opvideo/o/317/b.MP4.min.mp4">点我</a>
<img src="https://d2.example.com/x.jpg.min.avif" />
</body></html>`

const fixtureLoginWall = `<html><head><title>提示信息 - 测试论坛</title></head><body>请 登录 后查看</body></html>`

type stubDownloader struct{}

func customTestProfile() customProfile {
	return customProfile{ID: "custom:test", Name: "测试源", Base: "https://forum.example", SpaceUID: "680271", Cookie: "k=v"}
}

func TestCustomCatalogParsesSpacePage(t *testing.T) {
	d := &Downloader{}
	document, err := parseHTMLString(fixtureSpacePage)
	if err != nil {
		t.Fatal(err)
	}
	items, hasMore := customCatalogFromDocument(document, customTestProfile(), 1)
	if len(items) != 2 {
		t.Fatalf("catalog items = %d, want 2", len(items))
	}
	if items[0].ID != "custom:test:1001" || items[0].Title != "【视频】第一帖标题" {
		t.Fatalf("first item = %+v", items[0])
	}
	if !hasMore {
		t.Fatal("hasMore must follow the 下一页 anchor")
	}
	_ = d
}

func TestCustomDetailParsesObjectLinks(t *testing.T) {
	document, err := parseHTMLString(fixtureThreadPage)
	if err != nil {
		t.Fatal(err)
	}
	drama, chapters, err := customDetailFromDocument(fixtureThreadPage, document, customTestProfile(), "1001", "https://forum.example/forum.php?mod=viewthread&tid=1001")
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(chapters))
	}
	if chapters[0].Title != "第 6 集" {
		t.Fatalf("chapter 0 title = %q, want 第 6 集", chapters[0].Title)
	}
	if chapters[1].Title != "第 2 集" {
		t.Fatalf("chapter 1 title = %q, want 第 2 集 (fallback to order)", chapters[1].Title)
	}
	if !strings.Contains(chapters[0].VideoURL, "objectstorage") {
		t.Fatalf("chapter video url = %q", chapters[0].VideoURL)
	}
	if fmt.Sprint(drama.EpisodeCount) != "2" {
		t.Fatalf("episode count = %v", drama.EpisodeCount)
	}
}

func TestCustomDetailRejectsLoginWall(t *testing.T) {
	document, err := parseHTMLString(fixtureLoginWall)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := customDetailFromDocument(fixtureLoginWall, document, customTestProfile(), "1001", "https://forum.example/forum.php?mod=viewthread&tid=1001"); err == nil ||
		!strings.Contains(err.Error(), "Cookie") {
		t.Fatalf("login wall must report cookie hint, got %v", err)
	}
}

func TestCustomResolveDirectLink(t *testing.T) {
	media, err := customResolve(Chapter{VideoURL: "https://objectstorage.us-phoenix-1.oraclecloud.com/n/ns/b/opvideo/o/1/a.mp4", Referer: "https://forum.example/"})
	if err != nil || media.URL == "" {
		t.Fatalf("resolve = %+v %v", media, err)
	}
	if _, err := customResolve(Chapter{}); err == nil {
		t.Fatal("empty chapter must fail")
	}
	_ = context.Background()
}
