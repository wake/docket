# Design

## Context

Two packages look for programs, each with its own rules: `internal/gitx` (`lookPath`) finds git on the last `PATH` in its environment, resolving relative entries against the folder git runs in, and wants a regular file with an execute bit; `internal/openspec` (`resolve`, `isExecutable`) finds OpenSpec or the program in `DOCKET_OPENSPEC`, skipping relative `PATH` entries and resolving a relative program path against the folder it runs in, and wants anything but a folder with an execute bit. Both match the variable name `PATH` exactly. Neither uses `exec.LookPath`, since each looks in an environment given to it rather than the process's own, and each has rules `exec.LookPath` does not.

## Goals / Non-Goals

**Goals:**
- The same Windows rules in both packages, from one place, with their own other rules kept.
- Nothing changes on macOS or Linux.

**Non-Goals:**
- Running `.bat` or `.cmd` files differently from how Go's `os/exec` runs them.

## Decisions

### One small package for what Windows changes

`internal/execpath` holds three functions both packages call:

- `Getenv(env, key)`: the last value of `key` in `env`, comparing names without case on Windows, as Windows does.
- `Names(name, env)`: the file names to try for a program, in order: on Windows the name with each `PATHEXT` extension, a dot added where it is missing as `exec.LookPath` does, or the name alone when it already has one of them; elsewhere the name alone.
- `MayRun(fi)`: whether a file's permissions let it run: an execute bit, or always on Windows. Each caller keeps its own check of the kind of file.

Each function takes the system as an unexported parameter in an inner function, so its tests cover the Windows and the other rules on every system; the exported functions pass `runtime.GOOS`.

Alternative: `exec.LookPath`. It reads the process's own `PATH` and `PATHEXT`, not the environment dkt was given, and would drop the packages' rules for relative entries. Not chosen.

### Line endings: `.gitattributes`

`* text=auto eol=lf` makes every checkout write LF, whatever `core.autocrlf` says, so the embedded shipped files and their hashes are the same on every system. The repository holds no file that needs CRLF.

### Verification on Windows

The Windows CI job also runs `internal/execpath`, `internal/openspec`, `cmd/dkt`, the root package, the path tests of `internal/gitx` and the history tests of `internal/setup` (which compare the embedded files with the recorded hashes); the whole of `internal/gitx` and `internal/setup` still fails there for reasons `windows-suite` covers. The real OpenSpec test runs on Windows too, through `npx`, which npm installs as `npx.cmd`.

### Arguments to a batch file

Go hands a `.bat` or `.cmd` file to `CreateProcess`, which runs it through `cmd.exe`; `os/exec` quotes the arguments for `CommandLineToArgvW`, which `cmd.exe` does not follow, so `&`, `|`, `%` and others in an argument can run other commands. dkt passes OpenSpec names read from the repository (change folders, capability folders) and the user's `--tools`, so a cloned repository could hold a folder named to run a command. `Runner.exec` therefore checks, on Windows, when the program's extension is `.bat` or `.cmd` (compared without case), that every argument holds only ASCII letters, digits and `-_.,/@+=:`, and otherwise returns an `*UnsafeArgError` naming the argument without starting anything. Callers report it as they report any failed OpenSpec call (`dkt lint` as `openspec-failed`). The check is a function taking whether the system is Windows, so its tests run on every system.

Alternatives: quoting for `cmd.exe` with `SysProcAttr.CmdLine`, whose rules differ between `cmd.exe` and the batch file's own parsing and are easy to get wrong; running `node` with OpenSpec's script directly, which depends on how npm installed it. Not chosen; names with other characters are not change or capability names Docket makes.

## Risks / Trade-offs

- [A user's OpenSpec is a `.cmd` file and a name holds a character outside the set] → The call fails with a message naming the argument; the user can set `DOCKET_OPENSPEC` to run OpenSpec through `node` instead.
- [A folder on `PATH` holding both `git` and `git.exe` on Windows] → Windows itself would run `git.exe`; dkt does too, since it tries only names with an extension from `PATHEXT`.
