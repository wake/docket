---
name: docket
description: >-
  在這個 repository 開始任何開發工作時使用，要在腦力激盪、規劃或編輯之前：
  新功能、行為改變、bug 修正、重構，或對 prompt、測試、文件、CI 的編輯。
  不適用於不改任何東西的提問、說明或調查。
---

# Docket workflow

這是這個 repository 開發工作的規則，建立在 superpowers 與 OpenSpec 之上。
其他 skill 說法不同時，以這些規則為準。各條路徑的審查與驗證步驟不在這裡：
它們來自使用者的流程檔。

## Triage

開始前，用一行說明所走的路徑與選出它的規則。多條路徑都適用時，走最重的。
編輯依它碰到的東西分類：prompt 裡的錯字是 prompt 文字，搬移程式碼是重構。

- **Full review**（全審）：工作會改動 `openspec/specs/` 裡的一個句子。任何設下
  「沒有 spec 寫過、現有行為也沒顯示過」的預期的編輯也一樣，例如沒有明訂答案的
  bug 修正，或斷言新行為的測試：它定義了行為，由 change 加上那個句子。
- **Light review**（輕審）：
  - 還原已明訂行為的 bug 修正；
  - 不改變行為的重構（搬移程式碼算是）；
  - prompt 文字：agent 依其行動的文字，也就是 agent 依慣例載入的檔案
    （`AGENTS.md`、`CLAUDE.md`、skill 與 command 檔），以及程式交給 agent 的文字
    （內嵌或印出的指示、範本）；
  - 檢查已明訂或已顯示行為的測試；
  - 檢查設定，也就是決定跑哪些檢查的檔案，例如 CI 設定。
  會改變程式碼建置或執行方式的註解，例如編譯器指令（`//go:build`）或註記，算是程式碼。
- **Check**（查核）：本來會是直接合併、但碰到程式碼或測試檔的編輯，因為它只改了
  其中的註解或空白。
- **Direct merge**（直接合併）：給人看的文件（含其中的搬移與錯字）、只改程式碼或測試檔
  中的註解或空白且通過查核的編輯、版本號調整，以及只動依賴清單
  （如 `go.mod` 與 `go.sum`）的依賴升級。不開 change。

還沒準備好的 idea 放進 `docket/ideas/`，一個 idea 一個檔案。

在 change 或 record 的 worktree 裡，不要動 `docket/order.md` 與 `docket/ideas/` 底下的檔案。
工作中發現的 idea，以及對 order 的任何編輯，各自是一次直接合併。工作採用某個 idea 時，
`--source` 只放該 idea 的路徑（`docket/ideas/<file>.md`）。工作留下一部分待後續處理時，
把那一部分寫成新的 idea，其 `needs` 列出第一個 idea 的路徑，各自是一次直接合併。
要接下一項工作，執行 `dkt status --json`，取 `result.next` 所指那一行的 idea；
不要讀核取方塊，也不要讀 `docket/order.md` 的前幾行。

預期行為已經明訂（由 spec 的句子、文件或已出貨的行為），而現在不做的工作，是 issue，不是 idea：
在手上工作之外發現的 bug、最佳化、缺少的測試、別人提出的要求。專案的 `AGENTS.md` 有
`## Issues` 一節時，用它指名的工具寫 issue，段落用 `dkt issue template` 印出的那些；
先給使用者看，同意後才提交。否則執行
`dkt issue <name> --title "<one line>" --source "<where it was found>"` 並填寫各段落。
指不出預期行為在哪裡明訂時，它是 idea。

## The workflow file

分流之後、工作的任何其他步驟之前，執行 `dkt workflow --json`。它回報使用者的流程檔
（`result.path`）與其狀態（`result.status`）：

- `ok`：在下面每條路徑上，做檔案中該路徑標題下的步驟，以及這些步驟指向的 `## Reviews` 下的內容。
  在這些步驟上，檔案勝過使用者的其他指示。
- `missing` 或 `incomplete`：先執行 `docket-setup` skill；然後才開始工作。
- `stale`：照 `ok` 的方式依檔案做。在這個 session 裡告訴使用者一次：它是依較舊的範本做的、
  目前的範本（`dkt workflow template`）有哪些不同、以及 `docket-setup` skill 可以更新它。
- 其他任何情況，例如檔案無法讀取：告訴使用者 `dkt workflow` 回報了什麼，並問該怎麼繼續。
  不要寫入檔案。

## Full review

1. 在 git worktree 中開 change：
   `dkt new <slug> --kind feature|change-request|baseline --source "<where the request came from>" --title "<one line>"`。
   change 的 title、kind、source 與 base 在它的 `docket.yaml`；新 capability 的 delta 資料夾
   有帶其 title 的 `docket.yaml`。slug 已被使用時，`dkt new` 會用第一個空的後綴
   （`-2`、`-3`、...）建立 change 並回報；使用它回報的 slug。工作處理某個 issue 時，
   `--source` 只放該 issue：檔案 issue 的路徑（`docket/issues/<slug>.md`）或它在專案工具中的位址。
   工作採用某個 idea 時，`--source` 只放該 idea 的路徑。
2. superpowers:brainstorming 把它的 design 寫進 change 資料夾，而不是
   `docs/superpowers/specs/`：`proposal.md`、`specs/` 底下的 delta spec，以及需要時的
   `design.md`。每一份寫之前，先執行
   `openspec instructions <artifact> --change <slug> --json` 取得它的範本與規則。
3. superpowers:writing-plans 把 `openspec/changes/<slug>/plan.md` 寫成 plan，而不是
   `docs/superpowers/plans/`，依
   `openspec instructions plan --change <slug>`。沒有 tasks.md。task 是能單獨驗收的切片；
   耦合緊的 task 合為一個 task。需要審查者判斷的 task（錯誤路徑、依環境而定的行為、
   公開介面或輸出格式）寫 `**Review:** judgment`；其他寫 `**Review:** checks`。
4. 重用在 plan 裡決定：change 需要的每個 capability 在 plan 的 Reuse 表中都有一列，
   附上決定背後的搜尋。proposal 的 Builds On 點出已存在的東西。
5. 依流程檔 `## Full review` 下的步驟審查 plan、執行它，並在合併前審查 change。
   為某個 task 做的每個 commit 都帶 trailer `Change: <slug>` 與 `Task: <N>`。把它們放在
   訊息的最後一段，與其他 trailer（如 `Co-Authored-By`）放在一起、之間不留空行：
   git 只從最後一段讀 trailer。
   `.openspec.yaml` 指名其他 schema 的 change，依該 schema 的檔案走，除非轉換
   （見 `docket-doctor`）否則沒有 `plan.md`，仍用 `dkt archive` 封存。
6. 開 PR 之前，執行 `dkt sync <slug>` 與 `dkt lint <slug>`，並 commit plan.md。
7. 一個 change 是一個 PR，除非流程檔說要拆；它的 commit 呈現它的各層。PR 用 merge commit
   合併，絕不 squash：`dkt sync` 讀每個 commit 的 trailer。
8. 只在發版步驟中，於從預設分支開出的分支上用 `dkt archive` 封存。絕不直接執行
   `openspec archive`。

不要使用 OpenSpec 的 `/opsx:*` command 或 skill。

需求在中途改變時，依 plan 的規則：取消的 task、補償的 task、Revision 一節、Change Log 的一列。

## Light review

不開 change；`docket/records/` 裡的 record 說明這項工作做了什麼。在 git worktree 中：

1. 第一個 commit 之前，開 record：
   `dkt record <name> --kind fix|refactor|test|prompt|checks --source "<where the request came from>" --title "<one line>"`。
   kind 依分流：bug 修正（`fix`）、重構（`refactor`）、測試（`test`）、prompt 文字（`prompt`）、
   檢查設定（`checks`）。record 的 slug 是去掉 `.md` 的檔名：日期、名稱，名稱已被使用時再加後綴。
   使用 `dkt record` 回報的那個 slug，絕不用你傳入的裸名稱。工作處理某個 issue 時，
   `--source` 只放該 issue：檔案 issue 的路徑（`docket/issues/<slug>.md`）或它在專案工具中的位址。
   工作採用某個 idea 時，`--source` 只放該 idea 的路徑。
2. 依流程檔 `## Light review` 下的步驟做編輯。隨工作與其審查的進行填寫 record 的各段落，
   每段幾句話，取代掉註解。
3. 為這項工作做的每個 commit 都帶 trailer `Record: <slug>`。較大的工作可以用 plan 的格式在
   `## Tasks` 下列出 task；為其中一個做的 commit 也帶 `Task: <N>`。把它們放在訊息的最後一段，
   與其他 trailer（如 `Co-Authored-By`）放在一起、之間不留空行：git 只從最後一段讀 trailer。
4. record 列出 task 時，執行 `dkt sync <slug>` 並 commit record。執行 `dkt lint` 並修正它回報的
   問題。然後開 PR，用 merge commit 合併。

## Check

對只改註解或空白的程式碼或測試檔編輯：做流程檔 `## Check` 下的步驟，通過後照直接合併處理。

## Direct merge

在 git worktree 中：做編輯，做流程檔 `## Direct merge` 下的步驟，開 PR 並用 merge commit 合併。

## Parallel sessions

多個 session 平行工作時，每個 change 屬於一個 session，實作某個 change 的 session 不審查它。
開工前執行 `dkt conflicts`，並比對各 plan 的 `**Files:**` 區塊；重疊的 change 一個接一個執行。
依賴另一個 change 的 change 等它合併。一次合併一個 change：rebase 到預設分支、執行完整測試，
只在 rebase 有衝突之處重新審查。

## Prompt paths

agent 依其行動的文字列在 `AGENTS.md`。新增、搬移或移除其中一項的 PR，要在同一個 PR 更新那份清單。
