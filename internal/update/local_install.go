package update

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed embed/mhtodo.desktop embed/icon.png
var localPackaging embed.FS

// LocalInstallOptions controls InstallLocal (folder install into $PREFIX).
type LocalInstallOptions struct {
	// Prefix defaults to ~/.local when empty.
	Prefix string
	// SourceExecutable is the binary to copy; empty → os.Executable().
	SourceExecutable string
	// UnitPath overrides the systemd user unit path (tests).
	UnitPath string
}

// LocalInstallResult is the outcome of copying files into $PREFIX.
type LocalInstallResult struct {
	Prefix     string `json:"prefix"`
	Executable string `json:"executable"`
	Desktop    string `json:"desktop,omitempty"`
	Icon       string `json:"icon,omitempty"`
	UnitPath   string `json:"unit_path"`
	Message    string `json:"message"`
}

// DefaultPrefix returns ~/.local (or $HOME/.local).
func DefaultPrefix() string {
	home := userHome()
	if home == "" {
		return ".local"
	}
	return filepath.Join(home, ".local")
}

// InstallLocal copies this binary (and embedded desktop/icon) into $PREFIX,
// matching `make install` layout. Safe to run from an ephemeral go-build path —
// the copy lands under prefix; the source path is left alone.
func InstallLocal(opts LocalInstallOptions) (LocalInstallResult, error) {
	prefix := opts.Prefix
	if prefix == "" {
		prefix = DefaultPrefix()
	}
	prefix = filepath.Clean(prefix)

	src := opts.SourceExecutable
	if src == "" {
		exe, err := os.Executable()
		if err != nil {
			return LocalInstallResult{}, fmt.Errorf("resolve executable: %w", err)
		}
		src, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return LocalInstallResult{}, fmt.Errorf("resolve executable symlink: %w", err)
		}
	}

	destBin := filepath.Join(prefix, "bin", AppName)
	res := LocalInstallResult{
		Prefix:     prefix,
		Executable: destBin,
		UnitPath:   opts.UnitPath,
	}
	if res.UnitPath == "" {
		res.UnitPath = filepath.Join(userHome(), ".config", "systemd", "user", ServiceUnit)
	}

	if err := replaceFile(src, destBin, 0o755); err != nil {
		return res, fmt.Errorf("install binary: %w", err)
	}

	desktopDest := DesktopPath(prefix)
	if err := writeEmbeddedFile("embed/mhtodo.desktop", desktopDest, 0o644); err != nil {
		return res, fmt.Errorf("install desktop: %w", err)
	}
	res.Desktop = desktopDest
	_ = exec.Command("update-desktop-database", filepath.Join(prefix, "share", "applications")).Run()

	iconDest := IconPath(prefix)
	if err := writeEmbeddedFile("embed/icon.png", iconDest, 0o644); err != nil {
		return res, fmt.Errorf("install icon: %w", err)
	}
	res.Icon = iconDest
	res.Message = fmt.Sprintf("installed %s into %s", AppName, prefix)
	return res, nil
}

func writeEmbeddedFile(name, dest string, mode os.FileMode) error {
	in, err := localPackaging.Open(name)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return err
	}
	ok = true
	return nil
}

// InstallInfoForPrefix builds an InstallInfo pointing at $PREFIX/bin/mhtodo
// (used after InstallLocal so the systemd unit targets the installed binary).
func InstallInfoForPrefix(prefix, unitPath string) InstallInfo {
	prefix = filepath.Clean(prefix)
	exe := filepath.Join(prefix, "bin", AppName)
	if unitPath == "" {
		unitPath = filepath.Join(userHome(), ".config", "systemd", "user", ServiceUnit)
	}
	return InstallInfo{
		Executable: exe,
		Prefix:     prefix,
		UnitPath:   unitPath,
		HasService: serviceAttached(unitPath, exe),
		Arch:       goarch(),
	}
}
