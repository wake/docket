---
title: Docket 內的網頁 workbench（dkt serve）
---

2026-10-24 記下。

## 問題

進行中的 change 散在許多 worktree、分支與其他機器上。要讀或編輯 proposal、
delta 或 plan，得手動找到對的 worktree 資料夾。
團隊成員與不用終端機的人則完全做不到。

## 選定的方向

Docket 從同一個執行檔提供網頁 UI：`dkt serve`，也可以當作常駐的 daemon 執行
（launchd 或 systemd unit）。Docket 的掃描與讀取器就是引擎，
所以 UI 看到的和 CLI 看到的完全一樣：每個 change，來自每個 worktree、
本地分支與 remote-tracking 分支，並附上階段與 task 進度。

關於範圍的初步想法，都留待 brainstorming 時定案：

- 列出所有 change 與各自所在的位置；任何文件都可以渲染（表格、Mermaid）或看原始碼。
- 在 worktree 裡編輯文件，寫入前做比對檢查，讓編輯不會蓋掉 agent 剛才做的修改。
  只存在於分支上的 change 是唯讀。
- 從 UI 執行 Docket 指令（lint、conflicts、sync、archive）並顯示其 finding。
- 身分來自前面的 proxy（會傳遞使用者身分的存取閘道或私有網路）；
  Docket 只區分讀者與寫者。
- 可選擇包含 repository 裡的其他 markdown（`docs/`），不只 `openspec/`。

## 考慮過而擱置的替代方案

2026-10-24 調查；細節在維護者的研究筆記裡。

- Spectra 3.x（閉源）：只是 spec 目錄的檢視器與進度面板；
  看不到手動建立的 worktree；單一本機使用者。
- openspecui（MIT）：可編輯並偵測衝突，也會跟隨 `apply.tracks`，
  但只服務一個專案根目錄、綁定每個 OpenSpec 次版本，而且沒有使用者。
  若要 fork，就得一直追著它跑。
- spek（MIT）：唯一會彙整 worktree 的檢視器，但唯讀，
  而且寫死只認 `tasks.md`。
- SilverBullet（MIT）：成熟的多人 markdown 編輯，但完全不懂 OpenSpec，
  每個 worktree 要一個 space，而且讓每位寫者都能執行腳本。
- 內建進 agent session 管理器：對單一開發者可行，
  但這類工具通常只有單一管理者憑證，排除了「每個人」。

值得借用的點子：openspecui 的比對後寫入檔案寫入器、
spek 處理同一個 change 在各 worktree 間分歧的方式。

## 待釐清問題

- 第一版編輯做到多遠：只編輯文件，還是也要建立 change 與勾選 task。
- 在其他機器上尚未 push 的工作：每台機器一個 server 再加彙整檢視，還是只看已 push 的分支。
- UI 是放在這個 repository 出貨，還是另開一個 repository。
