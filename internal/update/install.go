package update

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Extracted holds paths to files pulled from a release tarball.
type Extracted struct {
	Dir     string // temp dir containing extracted files
	Binary  string
	Desktop string // may be empty
	Icon    string // may be empty
}

// ExtractTarball unpacks a release .tar.gz into a new temp directory.
// Layout is mhtodo_<ver>/mhtodo (+ optional .desktop, icon.png, README).
func ExtractTarball(tarPath string) (Extracted, error) {
	dir, err := os.MkdirTemp("", "mhtodo-update-*")
	if err != nil {
		return Extracted{}, err
	}
	ex := Extracted{Dir: dir}
	f, err := os.Open(tarPath)
	if err != nil {
		_ = os.RemoveAll(dir)
		return Extracted{}, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		_ = os.RemoveAll(dir)
		return Extracted{}, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.RemoveAll(dir)
			return Extracted{}, fmt.Errorf("tar: %w", err)
		}
		name := filepath.Clean(hdr.Name)
		// Strip the top-level mhtodo_<ver>/ component.
		if i := strings.IndexByte(name, '/'); i >= 0 {
			name = name[i+1:]
		}
		if name == "" || name == "." || strings.Contains(name, "..") {
			continue
		}
		dest := filepath.Join(dir, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0o755); err != nil {
				_ = os.RemoveAll(dir)
				return Extracted{}, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				_ = os.RemoveAll(dir)
				return Extracted{}, err
			}
			mode := os.FileMode(hdr.Mode) & 0o777
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				_ = os.RemoveAll(dir)
				return Extracted{}, err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				_ = os.RemoveAll(dir)
				return Extracted{}, err
			}
			if err := out.Close(); err != nil {
				_ = os.RemoveAll(dir)
				return Extracted{}, err
			}
			base := filepath.Base(name)
			switch base {
			case AppName:
				ex.Binary = dest
			case AppName + ".desktop":
				ex.Desktop = dest
			case "icon.png":
				ex.Icon = dest
			}
		}
	}
	if ex.Binary == "" {
		_ = os.RemoveAll(dir)
		return Extracted{}, fmt.Errorf("tarball missing %s binary", AppName)
	}
	return ex, nil
}

// InstallFiles copies the extracted binary (and desktop/icon when prefix is
// known) into place. Binary replacement uses write-temp + rename so a running
// process keeps its old inode on Linux.
func InstallFiles(ex Extracted, info InstallInfo) error {
	targetBin := info.Executable
	if err := replaceFile(ex.Binary, targetBin, 0o755); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	if info.Prefix == "" {
		return nil
	}
	if ex.Desktop != "" {
		dest := DesktopPath(info.Prefix)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := replaceFile(ex.Desktop, dest, 0o644); err != nil {
			return fmt.Errorf("install desktop: %w", err)
		}
		_ = exec.Command("update-desktop-database", filepath.Join(info.Prefix, "share", "applications")).Run()
	}
	if ex.Icon != "" {
		dest := IconPath(info.Prefix)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := replaceFile(ex.Icon, dest, 0o644); err != nil {
			return fmt.Errorf("install icon: %w", err)
		}
	}
	return nil
}

func replaceFile(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(dest)+".*")
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
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
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

// ServiceOps runs systemctl --user for stop / unit rewrite / enable --now.
// Overridable in tests.
type ServiceOps struct {
	Stop         func() error
	WriteUnit    func(unitPath, execStart string) error
	EnableNow    func() error
	DaemonReload func() error
	ImportEnv    func() error
}

// DefaultServiceOps talks to the real systemctl --user.
func DefaultServiceOps() ServiceOps {
	return ServiceOps{
		Stop: func() error {
			_ = exec.Command("systemctl", "--user", "stop", ServiceUnit).Run()
			return nil
		},
		WriteUnit: writeUnitFile,
		EnableNow: func() error {
			return runSystemctl("--user", "enable", "--now", ServiceUnit)
		},
		DaemonReload: func() error {
			return runSystemctl("--user", "daemon-reload")
		},
		ImportEnv: importGraphicalEnv,
	}
}

func writeUnitFile(unitPath, execStart string) error {
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return err
	}
	body := fmt.Sprintf(`[Unit]
Description=mhtodo — todo manager (GUI + system tray)
After=graphical-session.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
`, execStart)
	return os.WriteFile(unitPath, []byte(body), 0o644)
}

func runSystemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func importGraphicalEnv() error {
	for _, v := range []string{"DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR"} {
		if os.Getenv(v) == "" {
			continue
		}
		_ = exec.Command("systemctl", "--user", "import-environment", v).Run()
	}
	return nil
}

// ReinstallService stops the unit, rewrites it to ExecStart=<bin> gui, and
// enables it again.
func ReinstallService(ops ServiceOps, info InstallInfo) error {
	execStart := info.Executable + " gui"
	if ops.Stop != nil {
		if err := ops.Stop(); err != nil {
			return err
		}
	}
	if ops.WriteUnit != nil {
		if err := ops.WriteUnit(info.UnitPath, execStart); err != nil {
			return fmt.Errorf("write unit: %w", err)
		}
	}
	if ops.ImportEnv != nil {
		_ = ops.ImportEnv()
	}
	if ops.DaemonReload != nil {
		if err := ops.DaemonReload(); err != nil {
			return err
		}
	}
	if ops.EnableNow != nil {
		if err := ops.EnableNow(); err != nil {
			return err
		}
	}
	return nil
}
