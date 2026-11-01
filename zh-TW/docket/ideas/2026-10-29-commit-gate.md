---
title: 針對每個 commit 路徑的 PR 關卡（dkt check）
needs:
  - docket/ideas/2026-10-29-records.md
---

2026-10-29 記錄。以下各點已與維護者定案；採用這個 idea 的 change 會把它們寫成 proposal 與 delta。

## 問題

分流決定一件工作要走多少流程，但沒有任何東西檢查 commit 是否照著走：標成 docs 的 prompt 修改，或標成 chore 的結構調整，合併時都沒有經過審查。

## 已決定

- `dkt check` 檢視分支離開預設分支之後的每個非合併 commit，並在 CI 中作為必要檢查執行。
- `docket/config.yaml` 列出哪些路徑放程式碼、測試、prompt 和檢查設定（CI 設定與 `docket/config.yaml` 本身）。連同主 spec、change 資料夾與 record，這些路徑都受保護。
- 動到受保護路徑的 commit 帶一個相符的 trailer：主 spec（只有 change 會編輯）與該 change 的資料夾用 `Change: <slug>`；該 record 用 `Record: <slug>`；程式碼、測試、prompt 與檢查設定用 `Change:` 或 `Record:`。該 change 或 record 必須存在。
- 沒動到任何受保護路徑的 commit，其前綴是直接合併的類型之一時通過：`docs`、`style`、`build`、`chore`、`release`。
- 在程式碼與測試檔中，`docs:` 或 `style:` commit 不帶 trailer 也通過：一次編輯是否只改了註解或空白，不從內容判斷，因為那需要每種語言的語法，而且註解可能改變建置。分流會在合併前讓新開的 agent 確認；`dkt check` 不呼叫 agent，也看不到那件事是否發生。
- 其他 commit 一律失敗，並指出缺什麼。
- 沒有 `docket/config.yaml`，或其中沒有程式碼路徑時，`dkt check` 失敗，而不是全部放行。`dkt init` 會寫入範本。
- 封存 change 的 release commit，帶每個被封存 change 的 `Change:` trailer。
- 在 spec 定案前，先把規則跑過 Docket 自己的歷史，列出每個會失敗的 commit 及原因。

## 待釐清問題

- trailer 不在最後一段的 task commit，沒有 git 讀得到的 trailer，所以動到受保護路徑時關卡會讓它失敗。這是否就解決了 `docket/ideas/2026-10-24-stray-trailers.md`。
