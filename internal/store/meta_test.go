package store

import (
	"context"
	"testing"
)

func TestMetaGetSet(t *testing.T) {
	repo := openTestRepo(t)
	ctx := context.Background()

	if _, ok, err := repo.GetMeta(ctx, MetaAlwaysOnTop); err != nil || ok {
		t.Fatalf("absent key: ok=%v err=%v", ok, err)
	}

	if err := repo.SetMeta(ctx, MetaAlwaysOnTop, "true"); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, ok, err := repo.GetMeta(ctx, MetaAlwaysOnTop)
	if err != nil || !ok || v != "true" {
		t.Fatalf("get after set: v=%q ok=%v err=%v", v, ok, err)
	}

	if err := repo.SetMeta(ctx, MetaAlwaysOnTop, "false"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	v, ok, err = repo.GetMeta(ctx, MetaAlwaysOnTop)
	if err != nil || !ok || v != "false" {
		t.Fatalf("get after overwrite: v=%q ok=%v err=%v", v, ok, err)
	}

	// schema_version must still be readable (same table, different keys).
	sv, ok, err := repo.GetMeta(ctx, "schema_version")
	if err != nil || !ok || sv == "" {
		t.Fatalf("schema_version: v=%q ok=%v err=%v", sv, ok, err)
	}
}
