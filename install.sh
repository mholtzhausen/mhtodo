#!/usr/bin/env bash
# =============================================================================
# install.sh — one-shot installer for mhtodo (CLI + Wails GUI over SQLite).
#
# Pipe to bash:   curl -fsSL <url> | bash
#
# What it does:
#   1. Fetches the mhtodo source (SSH clone on this box, or HTTPS tarball).
#   2. Asks you: install as a *folder app* (binary on PATH) or as a *service*
#      (user systemd unit that launches it at login). Pass --service / --app to
#      skip the prompt when piping.
#   3. Builds via `make` and installs. If mhtodo is ALREADY installed, it updates
#      in place — folder update re-copies files; service update replaces + restarts.
#
# Targets used under the hood: make install (folder) / make service-install (unit).
# =============================================================================
set -euo pipefail

APP="mhtodo"
PREFIX="${HOME}/.local"
BRANCH="main"
REPO_SSH_DEFAULT="git@github.com:mholtzhausen/mhtodo.git"

SERVICE=0   # install as a user systemd service?
APP_ONLY=0  # just a folder install (binary on PATH)?
FORCE=0     # always do a full fresh install, even if up-to-date
NO_BUILD=0  # reuse existing binary, skip `make build`
ACCEPT=0    # accept the default choice non-interactively (for piping)
HELP=0

usage() {
cat <<EOF
install.sh — install or update mhtodo on this machine.

Usage:
  curl -fsSL <url> | bash
  bash install.sh [--service|-s|--app|-a] [--force] [--no-build] [--prefix DIR] [--repo-url URL] [-y] [--help]

Choices (mutually exclusive; prompt if neither given and not -y):
  --service, -s     install as a user systemd service (launches at login)
  --app,    -a      just a folder app (binary on ~/.local/bin, launcher + icon)

Other flags:
  --force           always full fresh install (rebuild + reinstall), ignore detection
  --no-build        reuse the existing binary; only re-copy files (folder update)
  --prefix DIR      install into DIR instead of ~/.local (default)
  --repo-url URL    clone from a specific SSH/HTTPS repo URL (override default)
  -y, --yes         accept the default choice without prompting
  --help            this help

Detection: if mhtodo is already installed in the chosen mode, it updates in place.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service|-s)   SERVICE=1 ;;
    --app|-a)       APP_ONLY=1 ;;
    --force)        FORCE=1 ;;
    --no-build)     NO_BUILD=1 ;;
    --prefix)       PREFIX="$2"; shift ;;
    --prefix=*)     PREFIX="${1#*=}" ;;
    --repo-url)     REPO_SSH="$2"; shift ;;
    --repo-url=*)   REPO_SSH="${1#*=}" ;;
    -y|--yes)       ACCEPT=1 ;;
    --help|-h)      HELP=1 ;;
    *) echo "unknown option: $1" >&2; usage; exit 2 ;;
  esac
  shift
done

[[ "$HELP" -eq 1 ]] && { usage; exit 0; }

# color, if TTY and not NO_COLOR
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  C='\033[0;32m'; B='\033[0;1m'; R='\033[0;31m'; N='\033[0m'
else
  C=''; B=''; R=''; N=''
fi
say()   { printf "${B}${C}%s${N}\n" "$*"; }
warn()  { printf "${R}warning:${N} %s\n" "$*" >&2; }

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
    read -r -p "Choose [1/2] (default 1): " pick; pick="${pick:-1}"
    case "$pick" in
      2) SERVICE=1; APP_ONLY=0 ;;
      *) APP_ONLY=1; SERVICE=0 ;;
    esac
  fi
fi

# ---------- detect existing install ----------
BIN_PATH="$PREFIX/bin/$APP"
UNIT_FILE="$HOME/.config/systemd/user/$APP.service"
service_present() { [[ -f "$UNIT_FILE" || "$(systemctl --user show -p ActiveState "$APP.service" 2>/dev/null | cut -d= -f2)" != "inactive" ]]; }
bin_present()     { [[ -x "$BIN_PATH" ]]; }

say "mhtodo installer — mode: $([[ $SERVICE -eq 1 ]] && echo 'service' || echo 'folder app')"

# ---------- fetch source ----------
SRC="$(mktemp -d)"
say "fetching mhtodo source…"
if command -v gh >/dev/null 2>&1 && gh auth status github.com >/dev/null 2>&1; then
  # SSH clone works without a prompt here (gh owns the token) — best for a private repo
  git clone "$REPO_SSH" "$SRC/$APP"
else
  warn "gh not authenticated on this box; falling back to HTTPS tarball (repo may need to be public or a token)"
  gh tarball "$BRANCH" -o "$SRC/$APP.tar.gz" && tar xzf "$SRC/$APP.tar.gz" --strip-components=1 -C "$SRC/$APP"
fi

MAKE="make PREFIX=\"$PREFIX\""
cd "$SRC/$APP"

# ---------- install / update ----------
if [[ $SERVICE -eq 1 ]]; then
  if bin_present || service_present; then
    say "service already installed — updating in place (replace + restart)"
    [[ $FORCE -eq 1 ]] && say "(--force: doing a full fresh reinstall instead)"
    if [[ $FORCE -eq 1 ]]; then
      "$MAKE" service-install
    else
      "$MAKE" service-install   # stops, rewrites unit, rebuilds, enable --now
    fi
  else
    say "fresh service install — building + installing user systemd unit"
    "$MAKE" service-install
  fi
else
  if bin_present; then
    say "folder app already installed — updating in place"
    if [[ $FORCE -eq 1 ]]; then
      say "(--force: doing a full fresh reinstall instead)"
      "$MAKE" install
    elif [[ $NO_BUILD -eq 1 ]]; then
      "$MAKE" install-files       # reuse existing binary, just re-copy files
    else
      "$MAKE" install             # rebuild + re-copy into prefix
    fi
  else
    say "fresh folder install — building + installing to $PREFIX"
    if [[ $NO_BUILD -eq 1 ]]; then
      "$MAKE" install-files       # (no binary yet, so this effectively installs)
    else
      "$MAKE" install
    fi
  fi
fi

# ---------- summary ----------
echo
say "done."
if [[ $SERVICE -eq 1 ]]; then
  say "service: systemctl --user status $APP.service"
else
  say "binary:  command -v $APP   (or: $BIN_PATH)"
fi
say "DB lives at:  $(command -v "$APP") >/dev/null 2>&1 && "$APP" path || echo '$XDG_DATA_HOME/$APP/$APP.db'"

# tidy temp source
rm -rf "$SRC"
