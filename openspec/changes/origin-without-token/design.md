## Context

`identify` (`internal/cli/run.go`) reads the origin with `git config --get remote.origin.url` and puts it in `repo.origin` unchanged. The project index takes the same value through `projects.Origin` (`internal/projects/locate.go`), which parses it with `net/url` and, for `http` and `https`, clears the user information. When `url.Parse` fails (an invalid percent escape anywhere in the URL, or a space in the password), `projects.Origin` keeps the raw string, token and all.

## Goals / Non-Goals

**Goals:**
- One rule, in one function, for both places that print or store the origin.
- The user information goes whether or not the URL is well formed.

**Non-Goals:**
- Rewriting index entries written before this change (see Risks).

## Decisions

### `identify` calls `projects.Origin`

`identify` replaces its own `RemoteURL("origin")` call with `projects.Origin(c.Git)`, keeping its error mapping (`identifyError`). The cli package already imports `projects`.

Alternative: move the function to `gitx` as a method. Rejected: it moves code without changing behavior, and the index's tests already sit next to it.

### Remove the user information by its place, not by parsing

For a URL whose scheme is `http` or `https` (compared without regard to case), the authority is the text after `://` up to the first `/`, `?` or `#`, or the end. When it holds an `@`, everything up to and including the last `@` in it is removed. Every other byte is kept as it is.

This is how RFC 3986 and `net/url` split the authority: user information cannot hold an unescaped `@`, so the last one ends it. A password with a raw `@` is still removed whole.

Alternatives:
- Keep `url.Parse` and add a fallback for when it fails. Rejected: two paths for one rule, and `u.String()` may also re-encode other parts of the URL.
- Redact rather than remove (`https://***@host`). Rejected: the index already removes, and a placeholder looks like a login name.

## Risks / Trade-offs

- [An index entry stored before this change from a URL that did not parse still holds the token] → It is rare (the URL must also be malformed). The release notes say to run `dkt projects rm <path>` and then `dkt projects add <path>` for such an entry.
- [A script compared `repo.origin` with git's own value] → They differ only when the URL holds user information; the proposal marks this as breaking.
