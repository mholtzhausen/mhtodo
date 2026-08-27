package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.2.0", 0},
		{"v1.2.0", "1.2.0", 0},
		{"1.2.0", "1.2.1", -1},
		{"1.3.0", "1.2.9", 1},
		{"1.2", "1.2.0", 0},
		{"dev", "1.0.0", -1},
		{"1.0.0-rc.1", "1.0.0", 0}, // prerelease suffix stripped for compare
	}
	for _, c := range cases {
		if got := CompareSemver(c.a, c.b); got != c.want {
			t.Errorf("CompareSemver(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestServiceAttachedAndEphemeral(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin", AppName)
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	unit := filepath.Join(t.TempDir(), ServiceUnit)
	if err := writeUnitFile(unit, bin+" gui"); err != nil {
		t.Fatal(err)
	}
	if !serviceAttached(unit, bin) {
		t.Fatal("expected attached")
	}
	if serviceAttached(unit, filepath.Join(dir, "other")) {
		t.Fatal("other exe should not attach")
	}
	if !IsEphemeralInstall("/tmp/go-build123/b001/exe/mhtodo") {
		t.Fatal("go-build should be ephemeral")
	}
	if IsEphemeralInstall(bin) {
		t.Fatal("real install should not be ephemeral")
	}
}

func TestLatestReleaseAndDownload(t *testing.T) {
	const ver = "9.9.9"
	asset := AssetName(ver, "amd64")
	payload, err := buildTestTarball(t, "fake-bin-contents")
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/mholtzhausen/mhtodo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		sum := sha256.Sum256(payload)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v" + ver,
			"html_url": "https://example.test/releases/v" + ver,
			"assets": []map[string]string{
				{
					"name":                 asset,
					"browser_download_url": "http://" + r.Host + "/download/" + asset,
					"digest":               "sha256:" + hex.EncodeToString(sum[:]),
				},
			},
		})
	})
	mux.HandleFunc("/download/"+asset, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(payload)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &GitHubClient{
		HTTP:      srv.Client(),
		OwnerRepo: OwnerRepo,
		APIBase:   srv.URL,
	}
	rel, err := client.LatestRelease("amd64")
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version != ver || rel.AssetName != asset {
		t.Fatalf("release: %+v", rel)
	}

	dest := filepath.Join(t.TempDir(), asset)
	if err := client.Download(rel.AssetURL, dest); err != nil {
		t.Fatal(err)
	}
	ex, err := ExtractTarball(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ex.Dir)
	b, err := os.ReadFile(ex.Binary)
	if err != nil || string(b) != "fake-bin-contents" {
		t.Fatalf("extracted binary: %q err=%v", b, err)
	}
	if ex.Desktop == "" || ex.Icon == "" {
		t.Fatalf("expected desktop+icon, got desktop=%q icon=%q", ex.Desktop, ex.Icon)
	}
}

func TestRunCheckAndUpdateWithService(t *testing.T) {
	const ver = "2.0.0"
	asset := AssetName(ver, "amd64")
	payload, err := buildTestTarball(t, "new-binary-v2")
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/mholtzhausen/mhtodo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		sum := sha256.Sum256(payload)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v" + ver,
			"html_url": "https://example.test/r",
			"assets": []map[string]string{{
				"name": asset, "browser_download_url": "http://" + r.Host + "/dl",
				"digest": "sha256:" + hex.EncodeToString(sum[:]),
			}},
		})
	})
	mux.HandleFunc("/dl", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prefix := t.TempDir()
	binDir := filepath.Join(prefix, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(binDir, AppName)
	if err := os.WriteFile(exe, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	unitPath := filepath.Join(t.TempDir(), ServiceUnit)

	var stopped, enabled bool
	ops := &ServiceOps{
		Stop: func() error { stopped = true; return nil },
		WriteUnit: func(path, execStart string) error {
			if !strings.HasSuffix(execStart, " gui") {
				t.Errorf("execStart=%q", execStart)
			}
			return writeUnitFile(path, execStart)
		},
		DaemonReload: func() error { return nil },
		ImportEnv:    func() error { return nil },
		EnableNow:    func() error { enabled = true; return nil },
	}

	client := &GitHubClient{HTTP: srv.Client(), OwnerRepo: OwnerRepo, APIBase: srv.URL}
	detect := func() (InstallInfo, error) {
		return InstallInfo{
			Executable: exe,
			Prefix:     prefix,
			HasService: true,
			UnitPath:   unitPath,
			Arch:       "amd64",
		}, nil
	}

	check, err := Run(Options{
		CurrentVersion: "1.0.0",
		CheckOnly:      true,
		Client:         client,
		Detect:         detect,
	})
	if err != nil {
		t.Fatal(err)
	}
	if check.UpToDate || check.Updated || !strings.Contains(check.Message, "update available") {
		t.Fatalf("check: %+v", check)
	}

	res, err := Run(Options{
		CurrentVersion: "1.0.0",
		Client:         client,
		Detect:         detect,
		Service:        ops,
		WorkDir:        t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || res.LatestVersion != ver || !stopped || !enabled {
		t.Fatalf("update: %+v stopped=%v enabled=%v", res, stopped, enabled)
	}
	got, err := os.ReadFile(exe)
	if err != nil || string(got) != "new-binary-v2" {
		t.Fatalf("binary after update: %q err=%v", got, err)
	}
	if _, err := os.Stat(DesktopPath(prefix)); err != nil {
		t.Fatalf("desktop missing: %v", err)
	}
	if _, err := os.Stat(IconPath(prefix)); err != nil {
		t.Fatalf("icon missing: %v", err)
	}
	unitBody, err := os.ReadFile(unitPath)
	if err != nil || !strings.Contains(string(unitBody), exe+" gui") {
		t.Fatalf("unit: %s err=%v", unitBody, err)
	}

	// already current
	again, err := Run(Options{
		CurrentVersion: ver,
		Client:         client,
		Detect:         detect,
	})
	if err != nil || !again.UpToDate || again.Updated {
		t.Fatalf("up-to-date: %+v err=%v", again, err)
	}
}

func buildTestTarball(t *testing.T, binContents string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "mhtodo_9.9.9")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	files := map[string][]byte{
		AppName:              []byte(binContents),
		AppName + ".desktop": []byte("[Desktop Entry]\nName=mhtodo\n"),
		"icon.png":           []byte("PNG"),
		"README.md":          []byte("# mhtodo\n"),
	}
	for name, body := range files {
		mode := os.FileMode(0o644)
		if name == AppName {
			mode = 0o755
		}
		if err := os.WriteFile(filepath.Join(root, name), body, mode); err != nil {
			return nil, err
		}
	}
	outPath := filepath.Join(dir, "out.tar.gz")
	f, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(b)
		return err
	})
	if err != nil {
		f.Close()
		return nil, err
	}
	if err := tw.Close(); err != nil {
		f.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(outPath)
}
