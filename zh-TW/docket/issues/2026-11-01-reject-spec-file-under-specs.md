---
title: 拒絕直接放在 specs/ 底下的 spec 檔案
source: review of add-conflicts, 2026-10-24; moved from docket/ideas/2026-10-24-spec-file-under-specs.md
---

## Current

`delta.CapabilityPath`（internal/delta/capability.go:9）接受 `specs/spec.md`，並回傳 capability `spec.md`。lint、archive、status 與 conflicts 都會呼叫它，所以各自都把該檔案當成 delta spec。

## Expected

capability 路徑的形式是 `specs/<capability>/spec.md`。OpenSpec 會忽略直接放在 `specs/` 底下的檔案，而 `conflict-detection` 比對的是「capability path」的值；Docket 也應該忽略它。

## Unchanged

巢狀的 capability 路徑如 `account/login` 仍可運作；`specs/<name>/spec.md` 仍被接受。

## Found

`add-conflicts` 的審查，2026-10-24。main 上仍存在：函式只檢查 `specs/` 前綴與 `/spec.md` 後綴。
