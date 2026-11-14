---
title: "Name OpenSpec's skills instead of the /opsx commands OpenSpec 2.0 removed"
kind: prompt
source: "OpenSpec 2.0.0 removed the /opsx commands: https://github.com/Fission-AI/OpenSpec/releases/tag/v2.0.0"
---

## What

The docket skill (`shipped/skill/SKILL.md` and the two copies `dkt init` installs here) and the docket-doctor skill now name OpenSpec's skills, `openspec-propose`, `openspec-new-change`, `openspec-apply-change` and `openspec-archive-change`, instead of `/opsx:*`, keeping the old command in brackets where a user may still type it. This repository's OpenSpec files moved to OpenSpec 2.0's layout, and `AGENTS.md` lists them where they now are.

## Why

OpenSpec 2.0.0 removed the `/opsx:*` commands, so in a project on 2.0 the skill told agents to archive with a command that does not exist. OpenSpec 1.14 to 2.0 all install these skills (`openspec-new-change` only in OpenSpec's extended profile, as `/opsx:new` was), so the names work for every supported version.

## Verification

Checked mechanically: the shipped skill and its two installed copies are identical, `dkt inspect` reports nothing with OpenSpec 2.0.0, and `gofmt -l .`, `go vet ./...` and `go test ./...` pass. In scratch repositories, `dkt init` with OpenSpec 1.14.0, 1.16.0 and 2.0.0 installed `openspec-propose`, `openspec-apply-change` and `openspec-archive-change`, and `openspec update` from 1.16.0 to 2.0.0 left Docket's files alone. Not verified: an agent following the new wording on 2.0.

## Review

