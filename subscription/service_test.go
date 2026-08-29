package subscription

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"one-proxy/model"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
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

func TestZeroRefreshIntervalIsPersisted(t *testing.T) {
	setupTestDB(t)
	profile := &model.Profile{Name: "manual", URL: "https://example.com/sub", FetchMode: model.ProfileFetchModeCache, Token: "manual-token"}
	if err := profile.Insert(); err != nil {
		t.Fatal(err)
	}
	stored, err := model.GetProfileById(profile.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.RefreshIntervalMinutes != 0 {
		t.Fatalf("refresh interval = %d, want 0", stored.RefreshIntervalMinutes)
	}
}

func TestFetchErrorDoesNotExposeSourceURL(t *testing.T) {
	profile := &model.Profile{URL: "https://source.example/sub?token=secret-token"}
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network unavailable")
	})}
	_, err := NewService(client).FetchDirect(context.Background(), profile, "Shadowrocket/2")
	if err == nil {
		t.Fatal("fetch unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), "source.example") || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("fetch error exposed source URL: %v", err)
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
	cache, err := service.FetchAndCache(context.Background(), profile, "Shadowrocket/2.2.70")
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

func TestRefreshCachesUAResponsesAndPreservesAnyTLS(t *testing.T) {
	setupTestDB(t)
	var mu sync.Mutex
	hits := make(map[string]int)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		variant := VariantForUserAgent(r.UserAgent())
		mu.Lock()
		hits[variant]++
		mu.Unlock()
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

	profile := &model.Profile{Name: "test", URL: upstream.URL, FetchMode: model.ProfileFetchModeCache, RefreshIntervalMinutes: 60, Token: "token", Status: model.ProfileStatusEnabled}
	if err := profile.Insert(); err != nil {
		t.Fatal(err)
	}
	service := NewService(upstream.Client())
	if err := service.Refresh(context.Background(), profile); err != nil {
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
	mu.Lock()
	defer mu.Unlock()
	for _, variant := range variants {
		if hits[variant] != 1 {
			t.Fatalf("variant %q fetched %d times, want 1", variant, hits[variant])
		}
	}
}

func TestFailedRefreshKeepsLastGoodCache(t *testing.T) {
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

	profile := &model.Profile{Name: "test", URL: upstream.URL, FetchMode: model.ProfileFetchModeCache, RefreshIntervalMinutes: 60, Token: "token", Status: model.ProfileStatusEnabled}
	if err := profile.Insert(); err != nil {
		t.Fatal(err)
	}
	service := NewService(upstream.Client())
	if err := service.Refresh(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	storedAfterSuccess, err := model.GetProfileById(profile.Id)
	if err != nil {
		t.Fatal(err)
	}
	lastSuccess := storedAfterSuccess.LastFetchTime
	returnHTML = true
	if err := service.Refresh(context.Background(), profile); err == nil {
		t.Fatal("HTML refresh unexpectedly succeeded")
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
