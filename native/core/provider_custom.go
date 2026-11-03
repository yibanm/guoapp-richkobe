package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// 自定义源：Discuz 论坛空间页作目录、帖子页内的对象存储直链作分集。
// source id 形如 custom:<slug>，剧集 id 形如 custom:<slug>:<tid>。
const customSourcePrefix = "custom"

type customProfile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Base     string `json:"base"`
	SpaceUID string `json:"spaceUid"`
	Cookie   string `json:"cookie"`
}

type customStore struct {
	mu       sync.RWMutex
	profiles map[string]customProfile
}

func newCustomStore() *customStore {
	return &customStore{profiles: map[string]customProfile{}}
}

func (s *customStore) replace(profiles []customProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[string]customProfile, len(profiles))
	for _, profile := range profiles {
		if profile.ID == "" {
			continue
		}
		next[profile.ID] = profile
	}
	s.profiles = next
}

func (s *customStore) bySlug(slug string) (customProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.profiles[customSourcePrefix+":"+slug]
	return profile, ok
}

// splitCustomSource 解析 source 级 id（custom:<slug>）
func splitCustomSource(source string) (slug string, ok bool) {
	source = strings.TrimSpace(source)
	rest, found := strings.CutPrefix(source, customSourcePrefix+":")
	if !found || strings.TrimSpace(rest) == "" {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

// splitCustomDramaID 解析剧集级 id（custom:<slug>:<tid>）
func splitCustomDramaID(id string) (slug, tid string, ok bool) {
	rest, found := strings.CutPrefix(strings.TrimSpace(id), customSourcePrefix+":")
	if !found {
		return "", "", false
	}
	slug, tid, found = strings.Cut(rest, ":")
	if !found || slug == "" || tid == "" {
		return "", "", false
	}
	return slug, tid, true
}

func isCustomSource(source string) bool {
	_, ok := splitCustomSource(source)
	return ok
}

type providerTextCookieKey struct{}

func customCookieContext(ctx context.Context, cookie string) context.Context {
	if cookie == "" {
		return ctx
	}
	return context.WithValue(ctx, providerTextCookieKey{}, cookie)
}

var customEpisodeNumber = regexp.MustCompile(`视频\s*(\d+)`)

// customCatalog 抓空间页帖子列表作剧目录
func (d *Downloader) customCatalog(ctx context.Context, profile customProfile, page int) ([]Drama, bool, error) {
	if profile.Base == "" || profile.SpaceUID == "" {
		return nil, false, errors.New("自定义源配置不完整：需要论坛地址与空间 UID")
	}
	address := fmt.Sprintf("%s/home.php?mod=space&uid=%s&do=thread&view=me&from=space&page=%d",
		strings.TrimSuffix(profile.Base, "/"), url.QueryEscape(profile.SpaceUID), max(1, page))
	ctx = customCookieContext(ctx, profile.Cookie)
	document, _, err := d.fetchProviderPage(ctx, address, profile.Base+"/", "")
	if err != nil {
		return nil, false, err
	}
	items, hasMore := customCatalogFromDocument(document, profile, page)
	if len(items) == 0 {
		return nil, false, errors.New("自定义源目录为空：检查空间 UID 与 Cookie 是否有效")
	}
	return items, hasMore, nil
}

func parseHTMLString(body string) (*html.Node, error) {
	return html.Parse(strings.NewReader(body))
}

// customCatalogFromDocument 从空间页文档提取帖子目录（纯解析，供测试）
func customCatalogFromDocument(document *html.Node, profile customProfile, page int) ([]Drama, bool) {
	source := customSourcePrefix + ":" + slugOf(profile)
	var items []Drama
	seen := map[string]bool{}
	for _, anchor := range providerHTMLNodes(document, func(node *html.Node) bool {
		return node.Data == "a" && strings.Contains(providerHTMLAttr(node, "href"), "tid=")
	}) {
		link := providerHTMLAttr(anchor, "href")
		parsed, err := url.Parse(link)
		if err != nil {
			continue
		}
		tid := parsed.Query().Get("tid")
		if tid == "" || seen[tid] {
			continue
		}
		title := strings.TrimSpace(providerHTMLText(anchor))
		if title == "" || len([]rune(title)) < 2 {
			continue
		}
		seen[tid] = true
		items = append(items, Drama{
			ID:          providerDramaID(source, tid),
			Source:      source,
			SourceID:    tid,
			Title:       title,
			ChannelName: profile.Name,
		})
	}
	return items, pageHasNext(document, page)
}

func slugOf(profile customProfile) string {
	rest, _ := strings.CutPrefix(profile.ID, customSourcePrefix+":")
	return rest
}

func pageHasNext(document *html.Node, page int) bool {
	next := strconv.Itoa(page + 1)
	for _, anchor := range providerHTMLNodes(document, func(node *html.Node) bool { return node.Data == "a" }) {
		href := providerHTMLAttr(anchor, "href")
		text := strings.TrimSpace(providerHTMLText(anchor))
		if strings.Contains(href, "page="+next) || text == "下一页" || text == "›" || text == ">" {
			return true
		}
	}
	return false
}

var customObjectLink = regexp.MustCompile(`https?://objectstorage[^\s"'<>]+`)

// customDetail 抓帖子页，把对象存储直链按出现顺序转成分集
func (d *Downloader) customDetail(ctx context.Context, profile customProfile, tid string) (Drama, []Chapter, error) {
	address := fmt.Sprintf("%s/forum.php?mod=viewthread&tid=%s",
		strings.TrimSuffix(profile.Base, "/"), url.QueryEscape(tid))
	ctx = customCookieContext(ctx, profile.Cookie)
	body, err := d.fetchProviderText(ctx, address, profile.Base+"/")
	if err != nil {
		return Drama{}, nil, err
	}
	document, parseErr := parseHTMLString(body)
	if parseErr != nil {
		return Drama{}, nil, parseErr
	}
	return customDetailFromDocument(body, document, profile, tid, address)
}

// customDetailFromDocument 从帖子页提取对象存储直链作分集（纯解析，供测试）
func customDetailFromDocument(body string, document *html.Node, profile customProfile, tid, pageURL string) (Drama, []Chapter, error) {
	if strings.Contains(body, "请 登录") || strings.Contains(body, "提示信息") {
		return Drama{}, nil, errors.New("自定义源登录态失效：请在设置里更新 Cookie")
	}
	source := customSourcePrefix + ":" + slugOf(profile)
	title := customPageTitle(document)
	links := customObjectLink.FindAllString(body, -1)
	if len(links) == 0 {
		return Drama{}, nil, errors.New("该帖没有解析到视频直链")
	}
	seen := map[string]bool{}
	var chapters []Chapter
	index := 0
	for _, raw := range links {
		link := strings.TrimSpace(raw)
		if seen[link] {
			continue
		}
		seen[link] = true
		index++
		chapters = append(chapters, Chapter{
			ID:       providerChapterID(source, tid, strconv.Itoa(index)),
			Source:   source,
			Title:    customEpisodeTitle(body, link, index),
			VideoURL: link,
			PageURL:  pageURL,
			Referer:  profile.Base + "/",
		})
	}
	drama := Drama{
		ID:           providerDramaID(source, tid),
		Source:       source,
		SourceID:     tid,
		Title:        title,
		ChannelName:  profile.Name,
		EpisodeCount: json.Number(strconv.Itoa(len(chapters))),
	}
	return drama, chapters, nil
}

func customPageTitle(document *html.Node) string {
	for _, node := range providerHTMLNodes(document, func(node *html.Node) bool { return node.Data == "title" }) {
		text := strings.TrimSpace(providerHTMLText(node))
		if cut, _, found := strings.Cut(text, " - "); found {
			text = strings.TrimSpace(cut)
		}
		if text != "" {
			return text
		}
	}
	return ""
}

// customEpisodeTitle 集数命名优先级：链接后的锚文本「视频N」（Discuz 的
// <a href="URL">文本</a> 文本恒在 URL 之后，只向后看避免密集链接串号）→
// URL 文件名里的「视频N」→ 出现序号
func customEpisodeTitle(body, link string, index int) string {
	position := strings.Index(body, link)
	if position >= 0 {
		tail := body[position+len(link) : min(len(body), position+len(link)+80)]
		if match := customEpisodeNumber.FindStringSubmatch(tail); len(match) > 1 {
			return "第 " + match[1] + " 集"
		}
	}
	if match := customEpisodeNumber.FindStringSubmatch(decodeURLName(link)); len(match) > 1 {
		return "第 " + match[1] + " 集"
	}
	return "第 " + strconv.Itoa(index) + " 集"
}

func decodeURLName(link string) string {
	parsed, err := url.Parse(link)
	if err != nil {
		return link
	}
	segments := strings.Split(parsed.Path, "/")
	if len(segments) == 0 {
		return link
	}
	return segments[len(segments)-1]
}

// customResolve 直链免登录可流式播放，直接交给播放器
func customResolve(chapter Chapter) (providerMedia, error) {
	if chapter.VideoURL == "" {
		return providerMedia{}, errors.New("自定义源章节缺少播放地址")
	}
	return providerMedia{URL: chapter.VideoURL, Referer: chapter.Referer}, nil
}

func (d *Downloader) customProbe(ctx context.Context, profile customProfile) (string, error) {
	items, _, err := d.customCatalog(ctx, profile, 1)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("连通成功：解析到 %d 个帖子", len(items)), nil
}
