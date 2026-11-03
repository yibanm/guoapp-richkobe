package core

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
)

// customCatalogResult 把论坛空间页目录转成引擎目录结果
func (engine *nativeEngine) customCatalogResult(ctx context.Context, input nativeInput) (nativeCatalogResult, error) {
	profile, found, err := engine.customSources.profileForSource(input.Source)
	if err != nil {
		return nativeCatalogResult{}, err
	}
	if !found {
		return nativeCatalogResult{}, errors.New("自定义源不存在或已被删除")
	}
	items, hasMore, err := engine.downloader.customCatalog(ctx, profile, max(1, input.Page))
	if err != nil {
		return nativeCatalogResult{}, err
	}
	result := nativeCatalogResult{Page: max(1, input.Page), HasMore: hasMore, Items: []nativeDrama{}}
	for _, drama := range items {
		result.Items = append(result.Items, nativeDrama{
			ID:       drama.ID,
			Source:   drama.Source,
			SourceID: drama.SourceID,
			Title:    drama.Title,
			Cover:    customText(drama.Cover),
			Episodes: customEpisodeCount(drama.EpisodeCount),
		})
	}
	return result, nil
}

func customText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func customEpisodeCount(value any) int {
	switch typed := value.(type) {
	case json.Number:
		number, _ := typed.Int64()
		return int(number)
	case string:
		number, _ := strconv.Atoi(typed)
		return number
	}
	return 0
}

// customDetailResult 把帖子页视频直链转成分集
func (engine *nativeEngine) customDetailResult(ctx context.Context, source, sourceID string) (any, error) {
	profile, found, err := engine.customSources.profileForSource(source)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("自定义源不存在或已被删除")
	}
	raw, chapters, err := engine.downloader.customDetail(ctx, profile, sourceID)
	if err != nil {
		return nil, err
	}
	drama := nativeDrama{
		ID:       raw.ID,
		Source:   raw.Source,
		SourceID: raw.SourceID,
		Title:    raw.Title,
		Cover:    customText(raw.Cover),
		Episodes: len(chapters),
	}
	return map[string]any{"drama": drama, "chapters": chapters, "warning": ""}, nil
}

func (engine *nativeEngine) customSourcesList() (any, error) {
	profiles, err := engine.customSources.list()
	if err != nil {
		return nil, err
	}
	return map[string]any{"profiles": profiles}, nil
}

func (engine *nativeEngine) customSourcesSave(raw json.RawMessage) (any, error) {
	var doc struct {
		Profiles []customProfile `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, errors.New("自定义源配置格式无效")
	}
	for index := range doc.Profiles {
		if doc.Profiles[index].Base == "" || doc.Profiles[index].SpaceUID == "" {
			return nil, errors.New("自定义源需要论坛地址与空间 UID")
		}
		if doc.Profiles[index].ID == "" {
			return nil, errors.New("自定义源缺少编号")
		}
	}
	if err := engine.customSources.save(doc.Profiles); err != nil {
		return nil, err
	}
	return true, nil
}

func (engine *nativeEngine) customSourcesProbe(ctx context.Context, source string) (any, error) {
	profile, found, err := engine.customSources.profileForSource(source)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("自定义源不存在或已被删除")
	}
	message, err := engine.downloader.customProbe(ctx, profile)
	if err != nil {
		return nil, err
	}
	return map[string]any{"message": message}, nil
}
