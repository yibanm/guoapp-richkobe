package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// 自定义源配置独立存 custom_sources.json，与 settings.json 隔离：
// 老用户设置文件不含本字段，读写门不受影响。Cookie 只落本机磁盘。
const customSourcesFile = "custom_sources.json"

type customSourcesDoc struct {
	Schema   int             `json:"schema"`
	Profiles []customProfile `json:"profiles"`
}

type customSourceManager struct {
	mu       sync.Mutex
	directory string
	store    *customStore
	loaded   bool
}

func newCustomSourceManager(directory string, store *customStore) *customSourceManager {
	return &customSourceManager{directory: directory, store: store}
}

func (m *customSourceManager) path() string {
	return filepath.Join(m.directory, customSourcesFile)
}

func (m *customSourceManager) ensureLoaded() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loaded {
		return nil
	}
	doc, err := m.readDoc()
	if err != nil {
		return err
	}
	m.store.replace(doc.Profiles)
	m.loaded = true
	return nil
}

func (m *customSourceManager) readDoc() (customSourcesDoc, error) {
	var doc customSourcesDoc
	raw, err := os.ReadFile(m.path())
	if errors.Is(err, os.ErrNotExist) {
		return customSourcesDoc{Schema: 1}, nil
	}
	if err != nil {
		return doc, err
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return customSourcesDoc{Schema: 1}, nil
	}
	if doc.Schema != 1 {
		return customSourcesDoc{Schema: 1}, nil
	}
	return doc, nil
}

func (m *customSourceManager) list() ([]customProfile, error) {
	if err := m.ensureLoaded(); err != nil {
		return nil, err
	}
	doc, err := m.readDoc()
	if err != nil {
		return nil, err
	}
	// 回读时抹掉 Cookie 明文，只告诉界面「有没有」
	for index := range doc.Profiles {
		if doc.Profiles[index].Cookie != "" {
			doc.Profiles[index].Cookie = "••••"
		}
	}
	return doc.Profiles, nil
}

func (m *customSourceManager) save(profiles []customProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, err := m.readDoc()
	if err != nil {
		return err
	}
	byID := map[string]customProfile{}
	for _, existing := range current.Profiles {
		byID[existing.ID] = existing
	}
	for index := range profiles {
		// 界面回传掩码占位时保留磁盘里的真实 Cookie
		if profiles[index].Cookie == "••••" {
			if existing, ok := byID[profiles[index].ID]; ok {
				profiles[index].Cookie = existing.Cookie
			}
		}
		profiles[index].ID = strings.TrimSpace(profiles[index].ID)
		profiles[index].Base = strings.TrimSuffix(strings.TrimSpace(profiles[index].Base), "/")
	}
	doc := customSourcesDoc{Schema: 1, Profiles: profiles}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, m.path()); err != nil {
		return err
	}
	m.store.replace(profiles)
	m.loaded = true
	return nil
}

func (m *customSourceManager) profileForSource(source string) (customProfile, bool, error) {
	slug, ok := splitCustomSource(source)
	if !ok {
		return customProfile{}, false, nil
	}
	if err := m.ensureLoaded(); err != nil {
		return customProfile{}, false, err
	}
	profile, found := m.store.bySlug(slug)
	return profile, found, nil
}
