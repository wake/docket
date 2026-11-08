---
title: Keep a token in the origin URL out of every command's output
---

Every command's JSON envelope prints `repo.origin` as git has it (`cli-output`, Repository identity). A remote such as `https://<token>@github.com/org/app.git` puts the token in every `--json` result, which scripts, CI and other tools pass on and log. The project index already removes the user information from `http` and `https` origins before storing them (`project-index`, Origin recorded); `repo.origin` could do the same.

Found in the pre-merge review of `projects-index`, 2026-11-08. It changes behavior `cli-output` settles, so it is a change request.

## Open questions

- Remove the user information only from `http` and `https` URLs, as the index does, or also from other forms.
