---
title: "Install and update dkt from docket-init and docket-upgrade on Windows"
kind: prompt
source: "docket/ideas/windows-skill-install.md"
---

## What

docket-init (step 2), docket-upgrade (step 2) and the README give one PowerShell script for Windows: download, SHA-256 check (stop on a mismatch), trial run, then `%LOCALAPPDATA%\Programs\dkt\dkt.exe`. The user adds that folder to `Path`; the agent does not. docket-upgrade also uses `curl.exe` and `Get-Command`. macOS and Linux keep `install.sh`.

## Why

`install.sh` refuses a Windows shell, so a Windows user stopped at installing dkt.

## Verification

- On `windows-latest` (#195, #196), with a local release folder: the script in PowerShell 5.1, 7 and Git Bash; a mismatch; a running `dkt.exe`; an update; `Get-Command`; `curl.exe`; the full paths; GitHub's `checksums.txt`. The blocks match the script byte for byte.
- Not verified: ARM64, the download of a Windows executable from a real release (none yet), Codex on Windows, and the `Path` dialog.

## Review

Codex (gpt-5.6-sol, adversarial) approved with no findings: the checksum check, the temporary folder, the quoting for Git Bash, `PATH`, and the untouched session activity step.
