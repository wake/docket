---
title: Write documents in Traditional Chinese first
kind: prompt
source: 'maintainer, 2026-11-02: write documents in Traditional Chinese first, then translate them into English'
---

## What

The Project section of AGENTS.md: Docket is a development process that uses superpowers and OpenSpec together; documents are written first in Traditional Chinese under `zh-TW/`, then translated into English at their own path. Adds `zh-TW/` with a Chinese copy of the existing documents.

## Why

The maintainer reads the English specs and ideas with difficulty; the repository is public, so the English files stay where they are.

## Verification

Mechanical checks: no Simplified characters; spec Requirement and Scenario headings, WHEN counts, front matter keys and fixed headings match the English; the YAML parses; `go test ./...` passes.

## Review

<!-- filled after the review -->
