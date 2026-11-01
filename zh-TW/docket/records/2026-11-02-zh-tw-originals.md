---
title: 文件先用繁體中文寫
kind: prompt
source: 'maintainer, 2026-11-02: write documents in Traditional Chinese first, then translate them into English'
---

## What

AGENTS.md 的 Project 段：Docket 定位為併用 superpowers 與 OpenSpec 的開發流程；文件先在 `zh-TW/` 寫繁中原稿，確認後翻成英文寫回原路徑。新增 `zh-TW/` 與既有文件的繁中鏡射。

## Why

維護者讀英文規格與 idea 很吃力；repository 公開，英文版仍要留在原路徑。

## Verification

機械檢查：無簡體字；spec 的 Requirement／Scenario 標題、WHEN 數、front matter 鍵、固定標題與英文版一致；YAML 可解析；`go test ./...`。

## Review
