#!/usr/bin/env bash
# =============================================================================
# install.sh — one-shot installer for mhtodo (CLI + Wails GUI over SQLite).
#
# Pipe to bash:   curl -fsSL <url> | bash
#                 ... | bash -s -- --service      # skip the prompt, install as a service
#
# What it does:
#   1. Downloads the latest PREBUILT release tarball (no Go/wails/node needed).
#      If no release is reachable it falls back to cloning + `make`.
#   2. Asks how you want to run mhtodo — folder app (binary on PATH) or systemd
#      user service at login — and WAITS for input even when piped (reads the
#      tty, not stdin). Pass --service / --app to skip the prompt; -y accepts
#      the default choice.
#   3. Installs — or UPDATES IN PLACE if mhtodo is already installed: a folder
#      install re-copies files over the existing ones; a service install
#      replaces binary + unit and restarts it.
# =============================================================================
set -euo pipefail

APP="mhtodo"
OWNER_REPO="mholtzhausen/mhtodo"
REPO_SSH_DEFAULT="git@github.com:mholtzhausen/mhtodo.git"
PREFIX="${HOME}/.local"

SERVICE=0    # install as a user systemd service?
APP_ONLY=0   # just a folder install (binary on PATH)?
NO_BUILD=0   # source-build fallback only: reuse existing binary, re-copy files
ACCEPT=0     # accept the default choice without prompting
HELP=0
COLOR_OPT="" # auto | on | off

usage() {
cat <<'EOF'
install.sh — install or update mhtodo on this machine.

Usage:
  curl -fsSL <url> | bash
  bash install.sh [--service|-s|--app|-a] [--no-build] [--prefix DIR] [--repo-url URL] [-y] [--color|--no-color] [--help]

Choices (mutually exclusive; you are prompted if neither is given and not -y):
  --service, -s     install as a user systemd service (launches at login)
  --app,    -a      just a folder app (binary on ~/.local/bin, launcher + icon)

Other flags:
  --no-build        source-build fallback only — reuse existing binary, re-copy files
  --prefix DIR      install into DIR instead of ~/.local (default)
  --repo-url URL    clone from a specific SSH/HTTPS repo URL in the build-from-source path
  -y, --yes         accept the default choice without prompting
  --color/--no-color force terminal colors on/off (auto by default)
  --help            this help

Source: downloads the latest GitHub Release tarball for your architecture. If no
release is reachable it clones the repo and runs `make` (needs go, wails, webkit2gtk-4.1).
If mhtodo is already installed in the chosen mode, it updates in place.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service|-s)   SERVICE=1 ;;
    --app|-a)       APP_ONLY=1 ;;
    --no-build)     NO_BUILD=1 ;;
    --prefix)       PREFIX="$2"; shift ;;
    --prefix=*)     PREFIX="${1#*=}" ;;
    --repo-url)     REPO_SSH="$2"; shift ;;
    --repo-url=*)   REPO_SSH="${1#*=}" ;;
    -y|--yes)       ACCEPT=1 ;;
    --color)        COLOR_OPT="on" ;;
    --no-color)     COLOR_OPT="off" ;;
    --help|-h)      HELP=1 ;;
    *) echo "unknown option: $1" >&2; usage; exit 2 ;;
  esac
  shift
done

[[ "$HELP" -eq 1 ]] && { usage; exit 0; }

# ---------- colors (auto unless forced) ----------
case "$COLOR_OPT" in
  on)  C=$'\033[0;32m'; B=$'\033[0;1m'; R=$'\033[0;31m'; N=$'\033[0m' ;;
  off) C=''; B=''; R=''; N='' ;;
  *)   if [[ -t 1 && -z "${NO_COLOR:-}" && "${TERM:-x}" != "dumb" ]]; then
         C=$'\033[0;32m'; B=$'\033[0;1m'; R=$'\033[0;31m'; N=$'\033[0m'
       else C=''; B=''; R=''; N=''; fi ;;
esac
say()  { printf "${B}${C}%s${N}\n" "$*"; }
warn() { printf "${R}warning:${N} %s\n" "$*" >&2; }

# ---------- tty for the prompt (stdin is consumed by `curl | bash`) ----------
TTYFD=""
{ exec {TTYFD}<> /dev/tty; } 2>/dev/null || TTYFD=""
ask() { # $1 = default answer; result in REPLY_
  if [[ -n "$TTYFD" ]]; then read -r REPLY_ <&"$TTYFD"; else printf "(no tty — using default)\n"; fi
  REPLY_="${REPLY_:-$1}"
}

# ---------- resolve choice ----------
REPO_SSH="${REPO_SSH:-$REPO_SSH_DEFAULT}"
if [[ $SERVICE -eq 1 && $APP_ONLY -eq 1 ]]; then
  warn "both --service and --app set — defaulting to --service"
  APP_ONLY=0
elif [[ $SERVICE -eq 0 && $APP_ONLY -eq 0 ]]; then
  if [[ $ACCEPT -eq 1 ]]; then
    say "no choice given; assuming folder app install (--app)"
    APP_ONLY=1
  else
    echo
    say "How do you want to run mhtodo?"
    printf "  %s1%s) %sfolder app%s — binary on PATH (make install)\n" "$B" "$N" "$C" "$N"
    printf "  %s2%s) %sservice%s   — user systemd unit at login (make service-install)\n" "$B" "$N" "$C" "$N"
    ask "1" && true
    case "${REPLY_}" in
      2) SERVICE=1; APP_ONLY=0 ;;
      *) APP_ONLY=1; SERVICE=0 ;;
    esac
    printf "→ %s\n" "$([[ $SERVICE -eq 1 ]] && echo 'systemd service' || echo 'folder app')"
  fi
fi

# ---------- arch + github token (for private-repo releases) ----------
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)   ARCH=amd64 ;;
  aarch64|arm64)  ARCH=arm64 ;;
  *) echo "error: unsupported architecture '$ARCH'" >&2; exit 1 ;;
esac

TOKEN=""
if [[ -n "${GH_TOKEN:-}" ]]; then
  TOKEN="$GH_TOKEN"
elif command -v gh >/dev/null 2>&1; then
  # local keyring read (no API round-trip); empty if gh is not logged in
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi

latest_version() { # prints latest release version (no v), empty if unreachable
  [[ -n "$TOKEN" ]] || return 0
  curl -fsSL -H "Authorization: Bearer $TOKEN" \
    "https://api.github.com/repos/$OWNER_REPO/releases/latest" |
  sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/p' | head -n1 || true
}

# ---------- fetch source ----------
say "mhtodo installer — mode: $([[ $SERVICE -eq 1 ]] && echo 'service' || echo 'folder app'), arch: $ARCH"
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT
SRC_MODE=""; INST_DIR=""

VER="$(latest_version)"
if [[ -n "$VER" ]]; then
  URL="https://github.com/$OWNER_REPO/releases/download/v$VER/${APP}_${VER}_linux_${ARCH}.tar.gz"
  say "found release v$VER — downloading prebuilt $ARCH binary (no toolchain needed)…"
  if curl -fsSL ${TOKEN:+-H "Authorization: Bearer $TOKEN"} "$URL" -o "$WORK/$APP.tar.gz"; then
    SRC_MODE=release; INST_DIR="$WORK/app"
    mkdir -p "$INST_DIR" && tar xzf "$WORK/$APP.tar.gz" -C "$INST_DIR" --strip-components=1
  fi
fi
if [[ -z "$SRC_MODE" ]]; then
  if [[ -n "$VER" ]]; then warn "release download failed for v$VER"; else warn "no release reachable (not authenticated, or none published yet)"; fi
  say "falling back to building from source (needs go + wails + webkit2gtk-4.1)…"
  SRC_MODE=source; INST_DIR="$WORK/app"
  if command -v gh >/dev/null 2>&1 && [[ -n "$(gh auth token 2>/dev/null || true)" ]]; then
    git clone --quiet "$REPO_SSH" "$INST_DIR"   # SSH: works anywhere gh is logged in (private repo OK)
  else
    warn "gh not authenticated — cannot fetch the private repo. Set GH_TOKEN, run 'gh auth login', or make the repo public."
    exit 3
  fi
fi

# ---------- existing-install detection ----------
BIN_PATH="$PREFIX/bin/$APP"
UNIT_FILE="$HOME/.config/systemd/user/$APP.service"
bin_present()     { [[ -x "$BIN_PATH" ]]; }
service_present() { [[ -f "$UNIT_FILE" ]] || systemctl --user is-enabled "$APP.service" >/dev/null 2>&1; }

# ---------- install primitives (release-tarball path) ----------
install_files() { # $1 = dir containing mhtodo, mhtodo.desktop, icon.png — overwrites in place
  local d="$1"
  install -Dm755 "$d/$APP"            "$PREFIX/bin/$APP"
  install -Dm644 "$d/$APP.desktop"    "$PREFIX/share/applications/$APP.desktop"
  install -Dm644 "$d/icon.png"        "$PREFIX/share/icons/hicolor/512x512/apps/$APP.png"
  command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$PREFIX/share/applications" || true
}

install_service() { # (re)write the user unit and start it; restarts a running instance
  systemctl --user stop "$APP.service" 2>/dev/null || true
  mkdir -p "$(dirname "$UNIT_FILE")"
  cat > "$UNIT_FILE" <<EOF
[Unit]
Description=mhtodo — todo manager (GUI + system tray)
After=graphical-session.target

[Service]
Type=simple
ExecStart=$PREFIX/bin/$APP gui
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
EOF
  local v
  for v in DISPLAY WAYLAND_DISPLAY XDG_RUNTIME_DIR; do [ -n "${!v:-}" ] && systemctl --user import-environment "$v" || true; done
  systemctl --user daemon-reload
  systemctl --user enable --now "$APP.service"
}

# ---------- install / update ----------
if [[ $SERVICE -eq 1 ]]; then
  if service_present; then say "service already installed — updating in place (replace + restart)"; else say "fresh service install…"; fi
  if [[ $SRC_MODE == source ]]; then
    make -C "$INST_DIR" PREFIX="$PREFIX" service-install
  else
    install_files "$INST_DIR"
    install_service
  fi
else
  if bin_present; then say "folder app already installed — updating in place"; else say "fresh folder install into $PREFIX…"; fi
  if [[ $SRC_MODE == source ]]; then
    if bin_present && [[ $NO_BUILD -eq 1 ]]; then make -C "$INST_DIR" PREFIX="$PREFIX" install-files; else make -C "$INST_DIR" PREFIX="$PREFIX" install; fi
  else
    install_files "$INST_DIR"
  fi
fi

# ---------- summary ----------
echo
say "done."
"$BIN_PATH" --version 2>/dev/null | sed 's/^/    /' || true
if [[ $SERVICE -eq 1 ]]; then
  say "service: systemctl --user status $APP.service"
else
  say "launch:  mhtodo   (tray icon appears in the panel; also in your app menu)"
fi
