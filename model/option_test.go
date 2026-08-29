package model

import (
	"one-proxy/common"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitSessionSecretPersistsGeneratedSecret(t *testing.T) {
	originalDB := DB
	originalSecret := common.SessionSecret
	t.Cleanup(func() {
		DB = originalDB
		common.SessionSecret = originalSecret
	})
	t.Setenv("SESSION_SECRET", "")

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "options.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	DB = db
	if err = DB.AutoMigrate(&Option{}); err != nil {
		t.Fatal(err)
	}

	common.SessionSecret = "first-generated-secret"
	if err = InitSessionSecret(); err != nil {
		t.Fatal(err)
	}
	if common.SessionSecret != "first-generated-secret" {
		t.Fatalf("unexpected initial session secret %q", common.SessionSecret)
	}

	common.SessionSecret = "new-secret-after-restart"
	if err = InitSessionSecret(); err != nil {
		t.Fatal(err)
	}
	if common.SessionSecret != "first-generated-secret" {
		t.Fatalf("session secret was not restored from the database: %q", common.SessionSecret)
	}
}

func TestInitSessionSecretPrefersEnvironment(t *testing.T) {
	originalSecret := common.SessionSecret
	t.Cleanup(func() { common.SessionSecret = originalSecret })
	t.Setenv("SESSION_SECRET", "environment-secret")

	common.SessionSecret = "generated-secret"
	if err := InitSessionSecret(); err != nil {
		t.Fatal(err)
	}
	if common.SessionSecret != "environment-secret" {
		t.Fatalf("environment secret was not used: %q", common.SessionSecret)
	}
}
