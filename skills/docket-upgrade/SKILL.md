---
name: docket-upgrade
description: >-
  Runs only when the user invokes it. Upgrades Docket on this machine to the
  latest release, then adjusts the Docket files in the current project.
disable-model-invocation: true
---

# Docket upgrade

Upgrade `dkt` and the Docket plugin on this machine to the latest release, then adjust the Docket files in the current project.

0. With the user's consent, you may make the installs and writes below; otherwise skip that step and go on to the next.

1. Check the versions all at once and show them to the user:
   - `dkt --version`
   - the plugin: in Claude Code, `claude plugin list --json`, and read `version` of `docket@docket`
   - the latest release: `curl -fsSL https://api.github.com/repos/wake/docket/releases/latest`, and read `tag_name` (in Windows PowerShell, `curl` is another command: use `curl.exe`)

   When the latest release cannot be found, say why and stop. When there is a newer release, read the notes of each release after the current version in `https://api.github.com/repos/wake/docket/releases`, and tell the user in a few sentences what changed.

2. Update `dkt`, unless it is already the latest:
   - macOS and Linux: `curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh` (installs `~/.local/bin/dkt`)
   - Windows (the coding agent says the environment is Windows, or `uname -s` starts with `MINGW`, `MSYS` or `CYGWIN`): run the script below in PowerShell; in a POSIX shell such as Git Bash, run it inside the single quotes of `powershell -NoProfile -Command '...'`. It downloads the latest `dkt-windows-amd64.exe` or `dkt-windows-arm64.exe`, by `PROCESSOR_ARCHITECTURE`, and `checksums.txt`; when the SHA-256 does not match, it deletes the download and stops; only once the file runs does it put it at `%LOCALAPPDATA%\Programs\dkt\dkt.exe`. When it fails, tell the user the error; `dkt.exe` cannot be replaced while a dkt runs, so end that dkt and run the script again.

     ```powershell
     $ErrorActionPreference = "Stop"; $ProgressPreference = "SilentlyContinue"
     $base = "https://github.com/wake/docket/releases/latest/download"
     $arch = @{ AMD64 = "amd64"; ARM64 = "arm64" }[$env:PROCESSOR_ARCHITECTURE]
     if (-not $arch) { throw "no dkt for $env:PROCESSOR_ARCHITECTURE" }
     $name = "dkt-windows-$arch.exe"
     $dir = Join-Path $env:LOCALAPPDATA "Programs\dkt"
     $tmp = Join-Path $dir ".install-$PID"
     New-Item -ItemType Directory -Force $tmp | Out-Null
     try {
       Invoke-WebRequest "$base/$name" -OutFile "$tmp\$name" -UseBasicParsing
       Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt" -UseBasicParsing
       $want = foreach ($l in Get-Content "$tmp\checksums.txt") { $f = -split $l; if ($f[1] -eq $name) { $f[0] } }
       $got = (Get-FileHash "$tmp\$name" -Algorithm SHA256).Hash
       if (-not $want -or $got -ne $want) { throw "SHA-256 of $name does not match checksums.txt" }
       & "$tmp\$name" --version
       if ($LASTEXITCODE -ne 0) { throw "$name does not run on this machine" }
       try { Move-Item -Force "$tmp\$name" "$dir\dkt.exe" }
       catch { throw "could not replace $dir\dkt.exe; Windows keeps it while a dkt runs: $_" }
       & "$dir\dkt.exe" --version
     } finally {
       Remove-Item -Recurse -Force $tmp
     }
     ```

   - Run `dkt --version` again. When the version has not changed, find the one in use with `command -v dkt` (in PowerShell, `(Get-Command dkt).Source`), tell the user, and stop. On Windows, when the one in use is not `%LOCALAPPDATA%\Programs\dkt\dkt.exe`, ask the user to add that folder to `Path` ahead of the other, in "Edit environment variables for your account", or to remove the other; do not change `PATH` for them.

3. Update the plugin, unless it is already the latest:
   - In Claude Code: run `claude plugin marketplace update docket`, then `claude plugin update docket@docket`.
   - In another coding agent: tell the user to update the plugin themselves.

   These instructions are still the old version's; the new plugin takes effect once it is reloaded (`/reload-plugins` in Claude Code).

4. Go on only when all three hold: this is a git directory, it has a `docket/workflow.md` file, and `dkt` is the latest or the user agreed to update it. Otherwise skip the rest of this step, tell the user the project was not adjusted, and go to step 5.

   Run `dkt inspect --json`; it is a read-only command. When it exits with 2 or 4, it did not check properly and has no result: say what else it reported and go to step 5. Otherwise act on the result:
   - `schema-outdated`, `skill-outdated`, `guidance-outdated`: name the files that will be updated, run `dkt init --json` once the user agrees, and help reload the skills as its next steps say. When these files are committed, triage them by the docket skill.
   - Any other finding, or `unreadable`: list them and do not fix them here. Suggest that the user run docket-doctor; when the plugin was just updated, it has to be reloaded first.
   - None of these: tell the user the project is up to date.

5. Finally, tell the user what was updated, what was skipped, and what is left to do: reloading the plugin, committing, running docket-doctor when needed.
