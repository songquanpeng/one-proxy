package main

import (
	"net/http/httptest"
	"one-proxy/common"
	"testing"

	"github.com/gin-contrib/sessions/cookie"
)

func TestConfigureSessionStoreUsesOneYearPersistentCookie(t *testing.T) {
	store := cookie.NewStore([]byte("01234567890123456789012345678901"))
	configureSessionStore(store)

	request := httptest.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()
	session, err := store.New(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["username"] = "root"
	if err = store.Save(request, response, session); err != nil {
		t.Fatal(err)
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge != common.SessionMaxAge {
		t.Fatalf("expected Max-Age %d, got %d", common.SessionMaxAge, cookies[0].MaxAge)
	}
	if !cookies[0].HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}
}
