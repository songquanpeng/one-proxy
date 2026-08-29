package subscription

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"one-proxy/model"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func headersFor(userAgent string) http.Header {
	return http.Header{"User-Agent": []string{userAgent}}
}

func setupTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Profile{}, &model.ProfileCache{}); err != nil {
		t.Fatal(err)
	}
	model.DB = db
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
}

func TestVariantForUserAgent(t *testing.T) {
	cases := map[string]string{
		"Shadowrocket/2.2.70":              VariantShadowrocket,
		"clash-verge-rev/2.4.2":            VariantClash,
		"ClashMetaForAndroid/2.11.16.Meta": VariantClash,
		"mihomo/1.19":                      VariantClash,
		"curl/8":                           VariantDefault,
	}
	for ua, expected := range cases {
		if actual := VariantForUserAgent(ua); actual != expected {
			t.Fatalf("VariantForUserAgent(%q) = %q, want %q", ua, actual, expected)
		}
	}
}

func TestFetchErrorDoesNotExposeSourceURL(t *testing.T) {
	profile := &model.Profile{URL: "https://source.example/sub?token=secret-token"}
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network unavailable")
	})}
	_, err := NewService(client).FetchDirect(context.Background(), profile, headersFor("Shadowrocket/2"))
	if err == nil {
		t.Fatal("fetch unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), "source.example") || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("fetch error exposed source URL: %v", err)
	}
}

func TestFetchIsTransparentForEndToEndHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "client/1" || r.Header.Get("Authorization") != "Bearer client-secret" || r.Header.Get("Accept-Language") != "zh-CN" {
			t.Errorf("end-to-end request headers were not forwarded: %#v", r.Header)
		}
		for _, stripped := range []string{"Cookie", "If-None-Match", "X-Forwarded-For", "X-Remove"} {
			if r.Header.Get(stripped) != "" {
				t.Errorf("request header %s should have been stripped", stripped)
			}
		}
		w.Header().Set("X-Provider-Meta", "preserved")
		w.Header().Set("Set-Cookie", "provider=secret")
		_, _ = fmt.Fprint(w, "opaque-subscription")
	}))
	defer upstream.Close()

	clientHeaders := headersFor("client/1")
	clientHeaders.Set("Authorization", "Bearer client-secret")
	clientHeaders.Set("Accept-Language", "zh-CN")
	clientHeaders.Set("Cookie", "one-proxy-session=secret")
	clientHeaders.Set("If-None-Match", "client-etag")
	clientHeaders.Set("X-Forwarded-For", "spoofed")
	clientHeaders.Set("Connection", "X-Remove")
	clientHeaders.Set("X-Remove", "connection-specific")

	cache, err := NewService(upstream.Client()).FetchDirect(context.Background(), &model.Profile{URL: upstream.URL}, clientHeaders)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cache.ResponseHeaders, "X-Provider-Meta") || strings.Contains(cache.ResponseHeaders, "Set-Cookie") {
		t.Fatalf("unexpected cached response headers: %s", cache.ResponseHeaders)
	}
}

func TestCacheMissCanFetchAndPersistCurrentUA(t *testing.T) {
	setupTestDB(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, r.UserAgent())
	}))
	defer upstream.Close()
	profile := &model.Profile{Name: "dynamic", URL: upstream.URL, FetchMode: model.ProfileFetchModeCache, Token: "dynamic-token"}
	if err := profile.Insert(); err != nil {
		t.Fatal(err)
	}
	service := NewService(upstream.Client())
	cache, err := service.FetchAndCache(context.Background(), profile, headersFor("Shadowrocket/2.2.70"))
	if err != nil {
		t.Fatal(err)
	}
	if cache.Variant != VariantShadowrocket || string(cache.Content) != "Shadowrocket/2.2.70" {
		t.Fatalf("unexpected dynamic cache: variant=%q content=%q", cache.Variant, cache.Content)
	}
	stored, err := service.Cached(profile.Id, "Shadowrocket/next-version")
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.Content) != "Shadowrocket/2.2.70" {
		t.Fatalf("dynamic cache was not persisted: %q", stored.Content)
	}
}

func TestClientRequestsCacheUAResponsesAndPreserveAnyTLS(t *testing.T) {
	setupTestDB(t)
	hits := make(map[string]int)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		variant := VariantForUserAgent(r.UserAgent())
		hits[variant]++
		w.Header().Set("Subscription-Userinfo", "upload=1; download=2; total=3")
		switch variant {
		case VariantClash:
			_, _ = fmt.Fprint(w, "proxies:\n  - name: node\n    type: anytls\n    server: example.com\n    port: 443\n    password: secret\n")
		case VariantShadowrocket:
			_, _ = fmt.Fprint(w, "anytls://secret@example.com:443#node")
		default:
			_, _ = fmt.Fprint(w, "opaque-default-subscription")
		}
	}))
	defer upstream.Close()

	profile := &model.Profile{Name: "test", URL: upstream.URL, FetchMode: model.ProfileFetchModeCache, Token: "token", Status: model.ProfileStatusEnabled}
	if err := profile.Insert(); err != nil {
		t.Fatal(err)
	}
	service := NewService(upstream.Client())
	if hits[VariantClash]+hits[VariantShadowrocket]+hits[VariantDefault] != 0 {
		t.Fatal("source was fetched without a client request")
	}
	if _, err := service.FetchAndCache(context.Background(), profile, headersFor("ClashMetaForAndroid/2.11.16.Meta")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.FetchAndCache(context.Background(), profile, headersFor("Shadowrocket/2.2.70")); err != nil {
		t.Fatal(err)
	}

	clash, err := service.Cached(profile.Id, "ClashMetaForAndroid/2.11.16.Meta")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(clash.Content), "type: anytls") {
		t.Fatalf("Clash payload was changed: %s", clash.Content)
	}
	shadowrocket, err := service.Cached(profile.Id, "Shadowrocket/2.2.70")
	if err != nil {
		t.Fatal(err)
	}
	if string(shadowrocket.Content) != "anytls://secret@example.com:443#node" {
		t.Fatalf("Shadowrocket payload was changed: %s", shadowrocket.Content)
	}
	if shadowrocket.SubscriptionUserinfo == "" {
		t.Fatal("subscription metadata header was not cached")
	}
	if hits[VariantClash] != 1 || hits[VariantShadowrocket] != 1 || hits[VariantDefault] != 0 {
		t.Fatalf("unexpected source fetches: %#v", hits)
	}
}

func TestFailedClientFetchKeepsLastGoodCache(t *testing.T) {
	setupTestDB(t)
	returnHTML := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if returnHTML {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, "<html>temporarily unavailable</html>")
			return
		}
		_, _ = fmt.Fprint(w, "last-good-subscription")
	}))
	defer upstream.Close()

	profile := &model.Profile{Name: "test", URL: upstream.URL, FetchMode: model.ProfileFetchModeCache, Token: "token", Status: model.ProfileStatusEnabled}
	if err := profile.Insert(); err != nil {
		t.Fatal(err)
	}
	service := NewService(upstream.Client())
	if _, err := service.FetchAndCache(context.Background(), profile, headersFor("Shadowrocket/2")); err != nil {
		t.Fatal(err)
	}
	storedAfterSuccess, err := model.GetProfileById(profile.Id)
	if err != nil {
		t.Fatal(err)
	}
	lastSuccess := storedAfterSuccess.LastFetchTime
	returnHTML = true
	if _, err := service.FetchAndCache(context.Background(), profile, headersFor("Shadowrocket/2")); err == nil {
		t.Fatal("HTML client fetch unexpectedly succeeded")
	}
	cache, err := service.Cached(profile.Id, "Shadowrocket/2")
	if err != nil {
		t.Fatal(err)
	}
	if string(cache.Content) != "last-good-subscription" {
		t.Fatalf("last good cache was overwritten: %s", cache.Content)
	}
	storedAfterFailure, err := model.GetProfileById(profile.Id)
	if err != nil {
		t.Fatal(err)
	}
	if storedAfterFailure.LastFetchTime != lastSuccess {
		t.Fatalf("last successful fetch time changed from %d to %d", lastSuccess, storedAfterFailure.LastFetchTime)
	}
}
