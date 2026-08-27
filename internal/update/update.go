package update

import (
	"fmt"
	"os"
	"path/filepath"
)

// Options controls a Run.
type Options struct {
	CurrentVersion string
	CheckOnly      bool // report only; do not download/install
	Force          bool // reinstall even when already current
	Client         *GitHubClient
	Detect         func() (InstallInfo, error) // nil → DetectInstall
	Service        *ServiceOps                 // nil → DefaultServiceOps when needed
	WorkDir        string                      // nil → os.MkdirTemp
}

// Result is the outcome of Run (also the --json envelope for the CLI).
type Result struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpToDate       bool   `json:"up_to_date"`
	Updated        bool   `json:"updated"`
	CheckOnly      bool   `json:"check_only,omitempty"`
	InstallPath    string `json:"install_path"`
	Prefix         string `json:"prefix,omitempty"`
	Service        bool   `json:"service"`
	Asset          string `json:"asset,omitempty"`
	ReleaseURL     string `json:"release_url,omitempty"`
	Message        string `json:"message"`
}

// Run checks GitHub for a newer release and optionally installs it.
func Run(opts Options) (Result, error) {
	detect := opts.Detect
	if detect == nil {
		detect = DetectInstall
	}
	client := opts.Client
	if client == nil {
		client = NewGitHubClient()
	}

	info, err := detect()
	if err != nil {
		return Result{}, err
	}
	if info.Arch != "amd64" && info.Arch != "arm64" {
		return Result{}, fmt.Errorf("unsupported architecture %q (need amd64 or arm64)", info.Arch)
	}
	if !opts.CheckOnly && IsEphemeralInstall(info.Executable) {
		return Result{}, fmt.Errorf("refusing to update ephemeral binary %s (install with make install / install.sh first)", info.Executable)
	}

	rel, err := client.LatestRelease(info.Arch)
	if err != nil {
		return Result{}, err
	}

	cur := NormalizeVersion(opts.CurrentVersion)
	res := Result{
		CurrentVersion: cur,
		LatestVersion:  rel.Version,
		InstallPath:    info.Executable,
		Prefix:         info.Prefix,
		Service:        info.HasService,
		Asset:          rel.AssetName,
		ReleaseURL:     rel.HTMLURL,
		CheckOnly:      opts.CheckOnly,
	}

	cmp := CompareSemver(cur, rel.Version)
	needsUpdate := cmp < 0 || opts.Force
	res.UpToDate = !needsUpdate
	if !needsUpdate {
		if cmp > 0 {
			res.Message = fmt.Sprintf("local v%s is newer than latest release v%s", cur, rel.Version)
		} else {
			res.Message = fmt.Sprintf("already up to date (v%s)", cur)
		}
		return res, nil
	}
	if opts.CheckOnly {
		if cmp < 0 {
			res.Message = fmt.Sprintf("update available: v%s → v%s", cur, rel.Version)
		} else {
			res.Message = fmt.Sprintf("would reinstall v%s (--force)", rel.Version)
		}
		return res, nil
	}

	work := opts.WorkDir
	cleanup := func() {}
	if work == "" {
		work, err = os.MkdirTemp("", "mhtodo-update-dl-*")
		if err != nil {
			return res, err
		}
		cleanup = func() { _ = os.RemoveAll(work) }
	}
	defer cleanup()

	tarPath := filepath.Join(work, rel.AssetName)
	if err := client.Download(rel.AssetURL, tarPath); err != nil {
		return res, err
	}
	if err := VerifyFileDigest(tarPath, rel.Digest); err != nil {
		return res, fmt.Errorf("verify download: %w", err)
	}

	ops := DefaultServiceOps()
	if opts.Service != nil {
		ops = *opts.Service
	}

	// Stop the GUI service before swapping the binary when one is attached.
	if info.HasService && ops.Stop != nil {
		_ = ops.Stop()
	}

	ex, err := ExtractTarball(tarPath)
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(ex.Dir)

	if err := InstallFiles(ex, info); err != nil {
		return res, err
	}

	if info.HasService {
		if err := ReinstallService(ops, info); err != nil {
			return res, fmt.Errorf("reinstall service: %w", err)
		}
	}

	res.Updated = true
	res.UpToDate = true
	res.CurrentVersion = rel.Version
	if info.HasService {
		res.Message = fmt.Sprintf("updated to v%s and restarted %s", rel.Version, ServiceUnit)
	} else {
		res.Message = fmt.Sprintf("updated to v%s", rel.Version)
	}
	return res, nil
}
