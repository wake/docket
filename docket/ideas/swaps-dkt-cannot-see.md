---
title: Guard dkt new against swaps it cannot see by path
---

The pre-merge review of #131 raised two races that writing through `os.Root` does not close. Both need another local process swapping folders inside the repository while `dkt` runs, and both were there before #131.

- OpenSpec runs as a subprocess and writes `openspec/changes/<slug>/` by path. A folder on that path swapped for a link while it runs sends its write outside the repository; `dkt new` refuses afterwards (`TestNewLinkAppearsWhileRunning`) but cannot undo it. Closing it would mean OpenSpec writing into a staging folder that `dkt` then moves in through the root, so `dkt`, not OpenSpec, would create the change folder, which `change-creation` and the issue fixed in #131 keep as it is.
- Cleanup in `dkt new` and `cleanupDated` in `dkt record` remove by name through the root: a plain folder another process swaps in at that path, holding the same names, is removed. Recording the folder's identity when it is made and comparing it before removing would close it. Both places carry a `shortcut:` comment.

Whether either is worth it depends on running `dkt` where processes that are not trusted can write the repository.
