---
title: "Test fakes that run without /bin/sh"
kind: test
source: "docket/ideas/windows-test-fakes.md"
---

## What

The `/bin/sh` fakes of OpenSpec and git are now copies of one Go program, `internal/testutil/fakebin`, built once per test process, each with a JSON payload at its end saying what it does, so a copy of a fake's bytes acts the same. `FakeOpenSpec` and `CountGit` keep their signatures; the new `FakeProgram` replaces the scripts in three tests. The Windows job runs `testutil`'s own tests.

## Why

The fakes could not run on Windows; `windows-runtime` needs them there.

## Verification

`go test ./...` passes on macOS, the other tests unchanged. Each mutation made the tests using a fake fail: OpenSpec's telemetry value, a relative PATH entry for git, an extra git run in `ReadBlobsAt`, an ignored origin read error, a fake left in place. Not verified on Windows: the tests using the fakes, until `windows-runtime`.

## Review

<!-- Who reviewed the diff (a model, or a person), what the review found,
and how each finding was handled. -->
