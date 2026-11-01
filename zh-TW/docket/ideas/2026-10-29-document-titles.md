---
title: 在 docket.yaml 中替 change 與 spec 命名標題
---

2026-10-29 記錄。以下各點已與維護者定案；採用這個 idea 的 change 會把它們寫成 proposal 與 delta。

## 問題

change 與主 spec 沒有讀者能用的標題：slug 是唯一的名稱。OpenSpec 忽略主 spec 的 H1，新建時寫 `# <id>
Specification`，`openspec show` 則把 id 當標題回報。`dkt status` 與 workbench 需要每一個的標題。

## 已決定

- change 的標題是 `openspec/changes/<slug>/docket.yaml` 中的 `title:`。`dkt new` 要求 `--title`。
- 主 spec 的標題是 `openspec/specs/<capability>/docket.yaml` 中的 `title:`。新增 spec 的 change 把標題放在
  `openspec/changes/<slug>/specs/<capability>/docket.yaml`，由 `dkt archive` 把它移到主 spec 旁，因為 OpenSpec 不會做。
- change 內的文件（proposal、design、plan、delta、evidence）不另設標題；它們依角色命名，掛在 change 的標題之下。
- `x-kind`、`x-source` 與 `x-base` 在同一個 change 中從 `.openspec.yaml` 搬到該 change 的 `docket.yaml`。已封存的 change 維持原樣，讀取端兩個位置都接受。
- 標題必填：`dkt lint` 回報沒有標題的進行中 change、新增 spec 的 delta 與主 spec。proposal 為現有的主 spec 草擬標題。
- `dkt status --json` 提供每個 change 的標題。
- Docket 把資料放在自己的檔案裡，所以 OpenSpec 的檔案維持 OpenSpec 寫出的樣子。以 OpenSpec 1.14.3 驗證過：change 根目錄或 spec 資料夾中的 `docket.yaml`，不影響 `validate`、`list`、`show`、`status`、`instructions` 與 `archive` 的運作，而放在 change 根目錄的那份會在封存時隨 change 一起移動。
