package subscription

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"one-proxy/common"
	"one-proxy/model"
)

const (
	VariantDefault      = "default"
	VariantClash        = "clash"
	VariantShadowrocket = "shadowrocket"
	MaxSubscriptionSize = 16 << 20
)

var variants = []string{VariantDefault, VariantClash, VariantShadowrocket}

var canonicalUserAgents = map[string]string{
	VariantDefault:      "one-proxy/1.0",
	VariantClash:        "clash-verge-rev/2 mihomo",
	VariantShadowrocket: "Shadowrocket/2",
}

type Service struct {
	client *http.Client
	locks  sync.Map
}

func NewService(client *http.Client) *Service {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Service{client: client}
}

var DefaultService = NewService(nil)

func ValidateProfile(profile *model.Profile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return errors.New("订阅名称不能为空")
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(profile.URL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("订阅链接必须是有效的 HTTP 或 HTTPS 地址")
	}
	if profile.FetchMode == "" {
		profile.FetchMode = model.ProfileFetchModeCache
	}
	if profile.FetchMode != model.ProfileFetchModeCache && profile.FetchMode != model.ProfileFetchModeProxy {
		return errors.New("未知的拉取模式")
	}
	if profile.RefreshIntervalMinutes < 0 {
		return errors.New("刷新间隔不能小于 0")
	}
	return nil
}

func VariantForUserAgent(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "shadowrocket") {
		return VariantShadowrocket
	}
	if strings.Contains(ua, "clash") || strings.Contains(ua, "mihomo") || strings.Contains(ua, "verge") {
		return VariantClash
	}
	return VariantDefault
}

func (s *Service) fetch(ctx context.Context, profile *model.Profile, variant string, userAgent string) (*model.ProfileCache, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profile.URL, nil)
	if err != nil {
		return nil, err
	}
	if userAgent == "" {
		userAgent = canonicalUserAgents[variant]
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := s.client.Do(req)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) {
			return nil, fmt.Errorf("请求源站失败: %v", urlError.Err)
		}
		return nil, fmt.Errorf("请求源站失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("源站返回 HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxSubscriptionSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxSubscriptionSize {
		return nil, fmt.Errorf("订阅内容超过 %d MiB 限制", MaxSubscriptionSize>>20)
	}
	if err := ValidatePayload(body, resp.Header.Get("Content-Type")); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	return &model.ProfileCache{
		ProfileId:             profile.Id,
		Variant:               variant,
		Content:               body,
		ContentType:           contentType,
		ContentDisposition:    safeContentDisposition(resp.Header.Get("Content-Disposition")),
		SubscriptionUserinfo:  resp.Header.Get("Subscription-Userinfo"),
		ProfileUpdateInterval: resp.Header.Get("Profile-Update-Interval"),
		ProfileWebPageURL:     resp.Header.Get("Profile-Web-Page-Url"),
		SupportURL:            resp.Header.Get("Support-Url"),
		ETag:                  fmt.Sprintf("\"%x\"", sum),
		FetchedTime:           time.Now().Unix(),
	}, nil
}

func safeContentDisposition(value string) string {
	if value == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}
	if filename := params["filename"]; filename != "" {
		return mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	}
	return ""
}

func (s *Service) FetchDirect(ctx context.Context, profile *model.Profile, userAgent string) (*model.ProfileCache, error) {
	return s.fetch(ctx, profile, VariantForUserAgent(userAgent), userAgent)
}

func (s *Service) FetchAndCache(ctx context.Context, profile *model.Profile, userAgent string) (*model.ProfileCache, error) {
	lockValue, _ := s.locks.LoadOrStore(profile.Id, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	variant := VariantForUserAgent(userAgent)
	cache, err := s.fetch(ctx, profile, variant, userAgent)
	if err != nil {
		_ = model.UpdateProfileFetchError(profile.Id, err.Error())
		return nil, err
	}
	if err = model.UpsertProfileCache(cache); err != nil {
		return nil, fmt.Errorf("保存缓存失败: %v", err)
	}
	_ = model.UpdateProfileFetchResult(profile.Id, cache.FetchedTime, "")
	return cache, nil
}

func (s *Service) Refresh(ctx context.Context, profile *model.Profile) error {
	lockValue, _ := s.locks.LoadOrStore(profile.Id, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	type result struct {
		variant string
		cache   *model.ProfileCache
		err     error
	}
	results := make(chan result, len(variants))
	var wg sync.WaitGroup
	for _, variant := range variants {
		wg.Add(1)
		go func(variant string) {
			defer wg.Done()
			cache, err := s.fetch(ctx, profile, variant, "")
			results <- result{variant: variant, cache: cache, err: err}
		}(variant)
	}
	wg.Wait()
	close(results)

	var failures []string
	successes := 0
	latestFetch := int64(0)
	for result := range results {
		if result.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", result.variant, result.err))
			continue
		}
		if err := model.UpsertProfileCache(result.cache); err != nil {
			failures = append(failures, fmt.Sprintf("%s: 保存缓存失败: %v", result.variant, err))
			continue
		}
		successes++
		if result.cache.FetchedTime > latestFetch {
			latestFetch = result.cache.FetchedTime
		}
	}
	fetchError := strings.Join(failures, "; ")
	if successes > 0 {
		_ = model.UpdateProfileFetchResult(profile.Id, latestFetch, fetchError)
	} else {
		_ = model.UpdateProfileFetchError(profile.Id, fetchError)
	}
	if len(failures) > 0 {
		return errors.New(fetchError)
	}
	return nil
}

func (s *Service) Cached(profileId int, userAgent string) (*model.ProfileCache, error) {
	wanted := VariantForUserAgent(userAgent)
	if cache, err := model.GetProfileCache(profileId, wanted); err == nil {
		return cache, nil
	}
	if wanted != VariantDefault {
		if cache, err := model.GetProfileCache(profileId, VariantDefault); err == nil {
			return cache, nil
		}
	}
	return model.GetProfileCache(profileId)
}

func (s *Service) RefreshDue(ctx context.Context) {
	profiles, err := model.GetEnabledCachedProfiles()
	if err != nil {
		common.SysError("读取待刷新订阅失败: " + err.Error())
		return
	}
	now := time.Now().Unix()
	for _, profile := range profiles {
		if profile.RefreshIntervalMinutes == 0 {
			continue
		}
		if profile.LastFetchError == "" && profile.LastFetchTime > 0 && now-profile.LastFetchTime < int64(profile.RefreshIntervalMinutes*60) {
			continue
		}
		if err := s.Refresh(ctx, profile); err != nil {
			common.SysError(fmt.Sprintf("刷新订阅 %s 失败: %v", profile.Name, err))
		}
	}
}

func StartRefresher(ctx context.Context) {
	go func() {
		DefaultService.RefreshDue(ctx)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				DefaultService.RefreshDue(ctx)
			}
		}
	}()
}

func ValidatePayload(body []byte, contentType string) error {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return errors.New("源站返回了空订阅")
	}
	lower := strings.ToLower(trimmed)
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType == "text/html" || strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html") {
		return errors.New("源站返回了 HTML 页面，不是订阅内容")
	}
	return nil
}
