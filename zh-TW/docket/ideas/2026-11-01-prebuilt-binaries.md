---
title: 每次發版附上預先建置的 dkt 執行檔
---

2026-11-01 記錄。維護者：要求使用者安裝 Go 並自己編譯 `dkt` 是錯誤的設計；發版應該附上能執行的執行檔。

## 問題

README 唯一的安裝方式是 `go install github.com/wake/docket/cmd/dkt@latest`，寫於 2026-10-24，之後沒有再檢視。發版沒有附加檔案（檢查過 v0.10.0 與 v0.11.0）。沒有 Go 的人無法安裝 `dkt`，而且每次安裝都要編譯。

## 方向，尚未決定

- 發版步驟為 macOS 與 Linux、arm64 與 amd64 建置 `dkt`，並把壓縮檔與它們的校驗碼附加到 GitHub release：GoReleaser，或在 tag 上執行的工作。
- 使用者用一個指令安裝：Homebrew tap、安裝腳本，或兩者都有。`go install` 留給從原始碼建置的人。
- `dkt --version`（`2026-10-24-dkt-version.md`）讓使用者與 `docket-doctor` 能判斷安裝的是哪個版本。
- Claude Code plugin 的 `bin/` 裡的檔案在 agent shell 的 PATH 上，所以 plugin 可以把 `dkt` 一起帶來
  （`2026-11-01-docket-init-command.md`）。

## 待釐清問題

- 哪些安裝管道，以及是否建置 Windows 版。
- 所選管道的 macOS 執行檔是否需要簽署與公證。
- 發版說明與封存步驟（`dkt archive`）如何與發版工作配合。

## 相關

- `2026-10-24-dkt-version.md`
- `2026-11-01-docket-init-command.md`
