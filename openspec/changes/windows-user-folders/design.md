# Design

## Context

All four user files are found by one function, `xdgFile` in `internal/projects/files.go`: an absolute `XDG_*` variable, else `HOME`, else `ErrNoHome`. It reads the environment it is given (`Options.Env`, which `cmd/dkt` fills from `os.Environ()` and tests build themselves), not the process's own, as the rest of dkt does. `testutil.Env` gives every test a temporary `HOME`; on Windows nothing sets `USERPROFILE` in it, so a test there would see none, and once dkt reads `USERPROFILE` it would see the runner's own if the test environment kept it.

## Goals / Non-Goals

**Goals:**
- One place decides which variable holds the home folder, and every user file goes through it.
- The Windows rule can be tested on any system.

**Non-Goals:**
- No change to how `dkt activity` or the hooks behave, beyond finding the folder.

## Decisions

### The variable name follows `os.UserHomeDir`, read from the given environment

`os.UserHomeDir` reads `USERPROFILE` on Windows and `HOME` elsewhere, but only from the process's environment. Calling it would make the user files the one thing dkt finds outside the environment it is given: tests would have to change the process environment (`t.Setenv`, which rules out parallel tests), and `RunWith` callers could not point dkt elsewhere. So dkt keeps reading its given environment, and takes from `os.UserHomeDir` only the rule: the variable is `USERPROFILE` on Windows and `HOME` elsewhere. A test checks that, for the process's own environment, the rule gives what `os.UserHomeDir` gives, so the two cannot drift apart.

Alternative: `HOME` when set and `USERPROFILE` otherwise on Windows, as the idea first proposed. Not taken: the maintainer named `USERPROFILE` as the home folder on Windows, and `os.UserHomeDir` does not read `HOME` there either.

### Names without case, through `execpath.Getenv`

`internal/projects` has its own `envValue`, which compares names exactly. It is replaced by `execpath.Getenv`, which already reads names without case on Windows and exactly elsewhere, so `UserProfile` or `xdg_state_home` work as Windows means them.

### The rule takes a `windows` parameter

As `internal/execpath` does, the lookup is an inner function taking `windows bool`, called with `runtime.GOOS == "windows"`; tests call it with `true` on macOS and Linux. The Windows CI job then runs the real paths end to end.

### The test environment sets both variables

`testutil.Env` sets `USERPROFILE` to the same temporary folder as `HOME`, on every system: on macOS and Linux dkt does not read it, and on Windows no test can reach the runner's own profile. Tests that remove the home folder to check `no-home` remove both.

## Risks / Trade-offs

- [A Windows user who sets `HOME` and expects dkt to use it] → dkt does not read it there, as Go does not; `XDG_CONFIG_HOME` and `XDG_STATE_HOME` remain the way to move the folders.
- [Git for Windows sets `HOME` for its own use] → it does not change where dkt looks, which stays the user's profile.
