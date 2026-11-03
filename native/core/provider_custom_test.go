package core

import "testing"

func TestCustomSourceIDs(t *testing.T) {
	if !isCustomSource("custom:shire") {
		t.Fatal("custom source not recognized")
	}
	if isCustomSource("hongguo") || isCustomSource("custom:") || isCustomSource("custom") {
		t.Fatal("non custom source misrecognized")
	}
	slug, tid, ok := splitCustomDramaID("custom:shire:342337")
	if !ok || slug != "shire" || tid != "342337" {
		t.Fatalf("drama id split = %q %q %v", slug, tid, ok)
	}
	if _, _, ok := splitCustomDramaID("custom:shire"); ok {
		t.Fatal("drama id without tid must be invalid")
	}
	if _, _, ok := splitProviderDramaID("custom:shire:342337"); !ok {
		t.Fatal("splitProviderDramaID must accept custom ids")
	}
}

func TestCustomEpisodeTitle(t *testing.T) {
	body := `前文 <a href="https://objectstorage.us-phoenix-1.oraclecloud.com/n/x/b/opvideo/o/316/%E8%A7%86%E9%A2%916-x.MP4.min.mp4">视频6-夹腿拍照</a> 后文`
	title := customEpisodeTitle(body, "https://objectstorage.us-phoenix-1.oraclecloud.com/n/x/b/opvideo/o/316/%E8%A7%86%E9%A2%916-x.MP4.min.mp4", 3)
	if title != "第 6 集" {
		t.Fatalf("episode title = %q", title)
	}
	fallback := customEpisodeTitle("无视频字样", "https://objectstorage.us-phoenix-1.oraclecloud.com/n/x/b/opvideo/o/9/a.mp4", 4)
	if fallback != "第 4 集" {
		t.Fatalf("fallback title = %q", fallback)
	}
}

func TestCustomStoreMaskKeepsCookie(t *testing.T) {
	directory := t.TempDir()
	store := newCustomStore()
	manager := newCustomSourceManager(directory, store)
	profiles := []customProfile{{ID: "custom:a", Name: "A", Base: "https://example.com/", SpaceUID: "1", Cookie: "secret=1"}}
	if err := manager.save(profiles); err != nil {
		t.Fatal(err)
	}
	listed, err := manager.list()
	if err != nil {
		t.Fatal(err)
	}
	if listed[0].Cookie != "••••" {
		t.Fatalf("list must mask cookie, got %q", listed[0].Cookie)
	}
	masked := []customProfile{{ID: "custom:a", Name: "A", Base: "https://example.com", SpaceUID: "1", Cookie: "••••"}}
	if err := manager.save(masked); err != nil {
		t.Fatal(err)
	}
	profile, found, err := manager.profileForSource("custom:a")
	if err != nil || !found || profile.Cookie != "secret=1" {
		t.Fatalf("masked save lost cookie: %+v %v %v", profile, found, err)
	}
}
