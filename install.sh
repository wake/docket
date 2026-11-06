#!/bin/sh
# Installs dkt from a GitHub release at ~/.local/bin/dkt.
# Everything is inside main, called on the last line: a shell reading this from
# curl runs each command as it arrives, so a download cut short must not hold
# a complete command that acts.
set -eu

die() {
	printf 'install.sh: %s\n' "$*" >&2
	exit 1
}

main() {
	releases=https://github.com/wake/docket/releases

	case $(uname -s) in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) die "unsupported system $(uname -s); see $releases" ;;
	esac
	case $(uname -m) in
	arm64 | aarch64) arch=arm64 ;;
	x86_64 | amd64) arch=amd64 ;;
	*) die "unsupported architecture $(uname -m); see $releases" ;;
	esac
	name=dkt-$os-$arch

	if command -v sha256sum >/dev/null 2>&1; then
		digest=sha256sum
	elif command -v shasum >/dev/null 2>&1; then
		digest="shasum -a 256"
	else
		die "neither sha256sum nor shasum is available to verify the download"
	fi

	case ${DKT_VERSION:-} in
	'') base=$releases/latest/download ;;
	v*) base=$releases/download/$DKT_VERSION ;;
	*) base=$releases/download/v$DKT_VERSION ;;
	esac

	case ${HOME:-} in
	/*) ;;
	*) die "HOME is not an absolute path" ;;
	esac

	dir=$HOME/.local/bin
	target=$dir/dkt
	[ ! -d "$target" ] || die "$target is a directory"
	mkdir -p "$dir"

	# Inside $dir so that mv replaces the target in one step, and not in /tmp,
	# which may be mounted noexec.
	tmp=$(mktemp -d "$dir/.dkt-install.XXXXXX")
	trap 'rm -rf "$tmp"' EXIT
	trap 'exit 1' HUP INT TERM

	for file in "$name" checksums.txt; do
		curl -fsSL "$base/$file" -o "$tmp/$file" || die "could not download $base/$file"
	done

	want=$(awk -v f="$name" '$2 == f { print $1 }' "$tmp/checksums.txt")
	[ -n "$want" ] || die "checksums.txt has no entry for $name"
	# shellcheck disable=SC2086
	got=$($digest <"$tmp/$name" | awk '{ print $1 }')
	[ "$got" = "$want" ] || die "SHA-256 of $name does not match checksums.txt"

	chmod 755 "$tmp/$name"
	version=$("$tmp/$name" --version) || die "$name does not run on this machine"
	mv -f "$tmp/$name" "$target"
	printf 'Installed %s at %s\n' "$version" "$target"

	case ":$PATH:" in
	*":$dir:"*) ;;
	*) printf '%s is not on your PATH; add it to run dkt.\n' "$dir" ;;
	esac
}

main
