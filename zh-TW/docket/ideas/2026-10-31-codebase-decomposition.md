---
title: 把既有程式碼拆解成 capability
---

2026-10-31 記錄，當時正在實作 `add-records`。維護者問 Docket 如何幫忙把既有專案的程式碼切成幾個部分，以及有哪些開源的軟體工程工具（不是 AI 工具）可以幫上忙。Docket 能不能做這件事，等採用這個 idea 時再研究；以下沒有任何一項已決定。

## 問題

在程式碼已存在之後才採用 Docket 的專案，有程式碼但沒有 spec。`baseline` 類型的 change 描述已經出貨的行為，但沒有東西幫忙判斷程式碼包含哪些 capability 與領域，或哪些檔案屬於哪個 capability。

## 既有做法

軟體工程把這件事稱為架構還原（architecture recovery），或軟體分群（software clustering）。它用四種訊號：

- **靜態相依**（import、呼叫）：哪段程式碼依賴哪段、循環、分層。例如：`go list -deps`；Emerge，它以 Louvain modularity 對相依圖分群，支援多種語言；JavaScript 的 dependency-cruiser；Java 的 ArchUnit 與 jQAssistant；設計結構矩陣工具如 Structure101、Lattix 與 Sonargraph，多半是商業軟體。
- **來自版本控制的演化耦合**：一起改動的檔案。例如：code-maat（Adam Tornhill）與它的 Python 移植版。
- **進入點**：CLI 指令、HTTP 路由、公開 API，常常對應到使用者看得到的 capability。
- **邊界檢查**：程式碼切開之後，避免各部分再長回一起：Go 的 arch-go 與 go-cleanarch、dependency-cruiser、Python 的 import-linter、ArchUnit。

Graphify 用 tree-sitter 把程式碼解析成可查詢的圖。解析是確定性的，但這個工具的對象是 AI 寫程式助理。

## 線索

- commit trailer 給了 Docket 單純 co-change 分析所缺的東西：`Change:` 與 `Record:` 指出 commit 屬於哪個 change 或 record，而 change 指出它的 capability。由此 `dkt` 可以回報每個 capability 隨時間動過哪些檔案。它只需要 git，而 Docket 本來就會讀 git，也適用於任何語言。
- 選擇 capability 與領域是判斷。它放在 prompt 文字裡（`baseline` 路徑、skill 或指令），絕不做成 `dkt` 的旗標，如 `2026-10-30-spec-domains.md` 對領域所定案的。
- 語言解析器與外部工具會把 Docket 綁在它們身上。任何使用都應是選用的，由使用者要求。

## 2026-11-01 已決定

與維護者一起調查過這些工具：在處理靜態結構、git 歷史與程式碼索引的工具中，多數已不再維護、只涵蓋一種語言，或需要自己的執行環境，而唯一能自動分群的那個不讀 Go。自動分群與人畫出的架構相比，分數也偏低。

- Docket 不為此使用任何外部工具。它只做自己能做的：git、檔案與 prompt 文字。
- 有證據的做法是介面清單。在 RepoReuse 預印本（arXiv 2609.35357）中，agent 重用既有程式碼的比例，有介面清單時是 67.8%，給完整原始碼是 29.2%，什麼都不給是 30.0%；介面那一組也告訴 agent 要重用，所以兩種效果沒有分開。它的條目包含模組、名稱、簽章、回傳契約與一行行為說明。

## 介面清單，尚未決定

沒有任何單一方法能在每種語言都給出簽章，而不用解析器函式庫或各語言自己的工具鏈：

- tree-sitter 最接近：一個執行環境，每種語言一份文法。它的 tag 查詢給的是名稱與位置，不是簽章。在 Go 中，官方綁定需要 cgo，純 Go 的執行環境尚未到 1.0，有幾份文法是 GPL 或 MPL。
- universal-ctags 只替 C 系語言補上簽章。
- 各語言自己的宣告輸出各有需求：TypeScript 與 Swift 要先建置，Python 的 `stubgen` 會 import 程式碼，rustdoc JSON 只有 nightly 才有。

方向：

- Go：`dkt` 用標準函式庫（`go/parser`、`go/doc`）建立清單。
- 其他語言：由 prompt 固定格式，agent 在 session 中從專案自己的工具鏈或讀程式碼來寫清單。行為那一行本來就需要判斷。`dkt` 只檢查格式以及清單是否過期。
- 先做一個小試驗，對象是一個既有的 package：不給地圖、給一份名稱清單，以及附上重用指示的介面清單。由試驗決定這份清單是否成為固定功能。

## 2026-10-31 試驗

對 Purdex repository 的一個 package（`internal/redact`，174 行與 5 個測試）寫出 baseline，由 Opus 在無人看管下透過 `docket` skill、在 dkt v0.9.1 上完成：

- 它花了 34 個回合、約 4 分鐘與 1.16 美元，寫了兩個 capability，共 17 條需求與 50 個情境，約 3,100 字。`openspec validate --strict` 與 `dkt lint` 都通過。測試沒涵蓋的地方，它用暫時的測試去探測程式碼，事後移除，並找到測試沒涵蓋的輸入。
- skill 與 schema 都沒有說明 baseline 該怎麼寫：多細、已知的缺陷是否照原樣寫成 spec、spec 的語言、capability 名稱、未解的問題放哪裡，以及如何不寫進 repository 就探測行為。隨附的 schema 還把它導向 `docs/ARCHITECTURE.md` 與 `dkt index`，而這兩樣只存在於 Docket 自己的 repository 與 plan 裡。
- 每行程式碼約對應 18 個字的 proposal 與 spec。Purdex 約有 390,000 行（240,000 行 Go、150,000 行 TypeScript）與約 34,000 個測試案例。照這個比率，整份 baseline 是數千次這樣的執行，spec 多到沒人審得完：baseline 必須界定範圍，依據是不靠 agent 做出的地圖。
- 那個 package 的地圖來自三個指令：`go doc -short`（三個函式與一個常數）、`go test -list`（五個測試）和用 `rg` 找它的 import 端（九個 package）。agent 最初的幾個步驟花在蒐集同樣的事實。
- baseline 路徑的兩個問題：已知的缺陷是否照原樣寫進 spec（這樣之後的修正會顯示成 spec 的改動），還是略過；以及 baseline 只到外部契約為止，還是也涵蓋規則的細節。

## 相關

- `2026-10-30-spec-domains.md`：spec 的領域。
- `2026-10-24-add-index.md`：程式碼匯出符號的索引。
- `2026-10-29-records.md`：`Record:` trailer。
- `2026-10-31-refactoring.md`：依這些事實規劃的結構調整。
- `2026-10-31-ub-mode.md`：可用來挑選要做 baseline 對象的指標。
- `2026-10-31-existing-project-adoption.md`：同一次試驗的其餘部分。

## 待釐清問題

- 在專案還沒有很多 change 之前，capability 與它們動過的檔案的報告是否有用，以及 Docket 之前沒有 trailer 的歷史如何涵蓋。
- 只靠歷史、沒有 trailer 的 co-change，是否值得在 `dkt` 中計算，以及在很長的歷史上如何維持低成本。
- `baseline` 路徑應要求 agent 用這些事實做什麼，以及是否需要自己的指令。
- 若有的話，要點名哪些外部工具作為選用的幫手，以及適用於哪些語言。
