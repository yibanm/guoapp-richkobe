package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// 真网络联调测试：仅当 CUSTOM_LIVE_COOKIE_FILE 指向 Netscape cookie 文件时运行，
// CI 无该变量自动跳过。Cookie 只从文件读，绝不写进仓库。
func TestCustomSourceLiveChain(t *testing.T) {
	cookieFile := os.Getenv("CUSTOM_LIVE_COOKIE_FILE")
	if cookieFile == "" {
		t.Skip("CUSTOM_LIVE_COOKIE_FILE not set")
	}
	raw, err := os.ReadFile(cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	var pairs []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 7 {
			pairs = append(pairs, parts[5]+"="+parts[6])
		}
	}
	cookie := strings.Join(pairs, "; ")
	proxyURL, _ := url.Parse("http://127.0.0.1:7897")
	d := &Downloader{
		cfg:          defaultConfig(),
		providerHosts: map[string]string{},
		proxyRouter:  &proxyRouter{},
		limiter:      newRequestLimiter(2, 300*time.Millisecond),
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	d.client = &http.Client{Transport: transport, Timeout: 45 * time.Second}
	profile := customProfile{
		ID: "custom:live", Name: "live", Base: "https://www.shireyishunjian.com/main",
		SpaceUID: "680271", Cookie: cookie,
	}
	engine, err := newNativeEngine(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	engine.downloader = d
	if err := engine.customSources.save([]customProfile{profile}); err != nil {
		t.Fatal(err)
	}

	t.Log("== step1 catalog")
	items, hasMore, err := d.customCatalog(context.Background(), profile, 1)
	if err != nil {
		t.Fatalf("catalog ERR: %v", err)
	}
	t.Logf("catalog ok: %d items hasMore=%v", len(items), hasMore)
	for i, item := range items {
		if i < 5 {
			t.Logf("  item %s | %s", item.SourceID, item.Title)
		}
	}
	if len(items) == 0 {
		t.Fatal("catalog empty")
	}

	t.Log("== step2 detail (first item with 视频 in title or first)")
	pick := items[0]
	for _, item := range items {
		if strings.Contains(item.Title, "视频") {
			pick = item
			break
		}
	}
	_, sourceID, ok := splitProviderDramaID(pick.ID)
	if !ok {
		t.Fatalf("drama id split fail: %s", pick.ID)
	}
	drama, chapters, err := d.customDetail(context.Background(), profile, sourceID)
	if err != nil {
		t.Fatalf("detail ERR (%s): %v", pick.Title, err)
	}
	t.Logf("detail ok: %s chapters=%d", drama.Title, len(chapters))
	for i, chapter := range chapters {
		if i < 4 {
			t.Logf("  ep %s %s", chapter.Title, chapter.VideoURL[:min(90, len(chapter.VideoURL))])
		}
	}
	if len(chapters) == 0 {
		t.Fatal("no chapters")
	}

	t.Log("== step3 resolve + HEAD")
	media, err := customResolve(chapters[0])
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodHead, media.URL, nil)
	resp, err := d.client.Do(req)
	if err != nil {
		t.Fatalf("head ERR: %v", err)
	}
	resp.Body.Close()
	t.Logf("head code=%d type=%s len=%s ranges=%s",
		resp.StatusCode, resp.Header.Get("Content-Type"),
		resp.Header.Get("Content-Length"), resp.Header.Get("Accept-Ranges"))
	if resp.StatusCode != 200 {
		t.Fatalf("media head not 200: %d", resp.StatusCode)
	}
	fmt.Println("LIVE CHAIN OK")
}
