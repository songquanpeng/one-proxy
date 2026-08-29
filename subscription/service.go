package subscription

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"one-proxy/model"
)

const (
	VariantDefault      = "default"
	VariantClash        = "clash"
	VariantShadowrocket = "shadowrocket"
	MaxSubscriptionSize = 16 << 20
)

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

func (s *Service) fetch(ctx context.Context, profile *model.Profile, variant string, clientHeaders http.Header) (*model.ProfileCache, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profile.URL, nil)
	if err != nil {
		return nil, err
	}
	copyClientHeaders(req.Header, clientHeaders)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}

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
	responseHeaders, err := json.Marshal(cacheableResponseHeaders(resp.Header))
	if err != nil {
		return nil, fmt.Errorf("保存响应头失败: %v", err)
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
		ResponseHeaders:       string(responseHeaders),
		ETag:                  fmt.Sprintf("\"%x\"", sum),
		FetchedTime:           time.Now().Unix(),
	}, nil
}

var strippedRequestHeaders = map[string]struct{}{
	"Connection":          {},
	"Cookie":              {},
	"If-Modified-Since":   {},
	"If-None-Match":       {},
	"Keep-Alive":          {},
	"Proxy-Authorization": {},
	"Proxy-Connection":    {},
	"Range":               {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
	"X-Forwarded-For":     {},
	"X-Forwarded-Host":    {},
	"X-Forwarded-Proto":   {},
}

func copyClientHeaders(destination http.Header, source http.Header) {
	connectionHeaders := make(map[string]struct{})
	for _, value := range source.Values("Connection") {
		for _, name := range strings.Split(value, ",") {
			connectionHeaders[http.CanonicalHeaderKey(strings.TrimSpace(name))] = struct{}{}
		}
	}
	for name, values := range source {
		canonicalName := http.CanonicalHeaderKey(name)
		if _, stripped := strippedRequestHeaders[canonicalName]; stripped {
			continue
		}
		if _, connected := connectionHeaders[canonicalName]; connected {
			continue
		}
		for _, value := range values {
			destination.Add(canonicalName, value)
		}
	}
}

var strippedResponseHeaders = map[string]struct{}{
	"Connection":          {},
	"Content-Disposition": {},
	"Content-Encoding":    {},
	"Content-Length":      {},
	"Content-Type":        {},
	"Keep-Alive":          {},
	"Proxy-Connection":    {},
	"Set-Cookie":          {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func cacheableResponseHeaders(source http.Header) http.Header {
	result := make(http.Header)
	for name, values := range source {
		canonicalName := http.CanonicalHeaderKey(name)
		if _, stripped := strippedResponseHeaders[canonicalName]; stripped {
			continue
		}
		result[canonicalName] = append([]string(nil), values...)
	}
	return result
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

func (s *Service) FetchDirect(ctx context.Context, profile *model.Profile, clientHeaders http.Header) (*model.ProfileCache, error) {
	return s.fetch(ctx, profile, VariantForUserAgent(clientHeaders.Get("User-Agent")), clientHeaders)
}

func (s *Service) FetchAndCache(ctx context.Context, profile *model.Profile, clientHeaders http.Header) (*model.ProfileCache, error) {
	lockValue, _ := s.locks.LoadOrStore(profile.Id, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	variant := VariantForUserAgent(clientHeaders.Get("User-Agent"))
	cache, err := s.fetch(ctx, profile, variant, clientHeaders)
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

func (s *Service) Cached(profileId int, userAgent string) (*model.ProfileCache, error) {
	wanted := VariantForUserAgent(userAgent)
	return model.GetProfileCache(profileId, wanted)
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
