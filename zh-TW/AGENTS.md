# AGENTS.md

給在此 repository 工作的 coding agent 的指示。Claude Code 透過 CLAUDE.md 讀取這份檔案。

## Project

Docket 是一套開發管理流程：併用 superpowers 與 OpenSpec 做 spec 驅動開發（SDD）。`dkt` 是它的 Go CLI，為 OpenSpec change 資料夾做檢查與記帳。見 README.md。這是公開的 repository：絕不 commit 主機名稱、客戶名稱、個人路徑或秘密。

文件先在 `zh-TW/` 用繁體中文寫原稿，維護者確認後翻成英文寫回原路徑，兩邊放同一個 PR；規則見 `zh-TW/README.md`。`zh-TW/` 與 `README.zh-TW.md` 以外的一切都用英文，程式碼、commit 與 PR 也是。

狀態：里程碑 1 已完成。`dkt init`、`dkt new`、`dkt record`、`dkt issue`、`dkt lint`、`dkt conflicts`、`dkt sync`、`dkt archive`、`dkt rebase`、`dkt status`、`dkt workflow` 與 `dkt inspect` 都已存在；用 `go install ./cmd/dkt` 建置。接下來的工作排在 `docket/order.md`。

撰寫 proposal 或 plan 之前，先讀 `docs/ARCHITECTURE.md`，了解模組邊界與共用的 capability。Idea 在 `docket/ideas/`，一個檔案一個；`docket/order.md` 依序列出排入佇列的 idea，而 `dkt status` 回報為下一項的那一行，就是下一件工作。

## Development

- Go module 在 repository 根目錄；`go vet ./...` 與 `go test ./...`。
- OpenSpec CLI 固定在 1.14.x。不安裝而使用固定版本的副本：`DOCKET_OPENSPEC="npx -y @fission-ai/openspec@1.14.3"`。
- `schema/docket/` 是 Docket 出貨的 schema（以 `go:embed` 嵌入）；`dkt init` 每台機器安裝一次，而不是裝進每個專案。對它的修改與其他檔案一樣走同樣的分流；它的指示與範本是出貨給其他專案的 prompt。
- Prompt 路徑，也就是 agent 據以行動的文字：`AGENTS.md`、`CLAUDE.md`、`openspec/config.yaml` 裡的 `context`、`schema/docket/`、`.claude/skills/docket/`、`.claude/skills/docket-setup/`、`.claude/skills/docket-doctor/`、`.agents/skills/` 底下同樣三個、`user-skills/docket-init/`、`workflow/template.md`、`record/template.md` 與 `issue/template.md`。`dkt init` 把 `schema/docket/`、`docket`、`docket-setup` 與 `docket-doctor` skill，以及 `AGENTS.md` 的 Docket 區塊出貨給其他專案（並在使用者層級安裝 `docket-init`），而 `dkt workflow template` 印出 `workflow/template.md`。

<!-- docket:start -->
## Docket

在此 repository 開始任何開發工作之前（新功能、行為改變、修 bug、重構，或修改 prompt、測試、文件或 CI），先使用 `docket` skill，再進行 brainstorming、規劃或編輯。

`docket` skill 與其他 skill 不一致時，以 `docket` skill 為準。特別是：brainstorming 把設計寫進 change 資料夾，而不是 `docs/superpowers/specs/`；plan 放在 `openspec/changes/<slug>/plan.md`，而不是 `docs/superpowers/plans/`；只有標了 `**Review:** judgment` 的 task 才有自己的審查者；而且不使用 OpenSpec 的 `/opsx:*` 命令。

每條路徑的步驟來自使用者的流程檔，由 `dkt workflow` 找到；即使使用者的其他指示不同，也照它做。除此之外，這個區塊不覆寫使用者自己的指示。
<!-- docket:end -->
