# Issues in another tool

When the project's `AGENTS.md` has a `## Issues` section, an issue's
content is kept in the tool it names, and Docket stands for it with a card:
an issue file in `docket/issues/` whose front matter adds `external` (the
issue's address) and `external-id` (the number people use, such as `#92`),
with no body (a dropped card adds the `## Dropped` section saying why). A
card's state and handlers come from the changes and records that take it,
as for any issue. Read and write the tool with the tool `## Issues` names;
below, syncing means reading the issue's content, comments and last update
time.

- **Making a card**: check that no card has the address, run
  `dkt issue <name> --external <address> --external-id <number> --title "<the tool's title>" --source "<source>"`,
  then run `dkt lint`. A new card is not added to the order on its own.
- **A new issue**: with the sections from `dkt issue template`, open it in
  the tool only once the user agrees, then make its card, with `--source`
  saying where it was found. When it cannot be opened, write a local issue
  with its sections and `unsynced: [open]`; once it can be opened, open it,
  add `external` and `external-id`, empty the body and remove `open`.
- **An issue already in the tool**: only make its card, with its address
  as `--source`; no proposal or record is written now.
- **The user asks to sync**: the user says which issues. Those without a
  card get one, as above. For those with a card, update `title` when the
  tool's title changed, and list for the user those the tool has closed
  that are not done, marking them `dropped` only once the user agrees. These
  edits are a direct merge of their own.
- **Before work starts**: sync before brainstorming or the record. Update
  the card when the tool's title changed; when the tool has closed the
  issue, stop and ask the user. Sum up the content and comments read, with
  the tool's last update time as read, in the proposal or the record. Edits
  to the card go into the work's first commit.
- **Before the pull request**: sync again, and add to the proposal or the
  record what was added since work started.

When the tool can be written to, leave a comment at these points: when a
change or record takes the card (which work), when its pull request is
opened (the pull request's address), when it is merged (the merge commit;
whether to close the issue as well follows `## Issues`), and when the card
is dropped (the reason). Make only the writes `## Issues` allows, and ask
the user before any other. When a comment cannot be written, add its point
(`taken`, `pr`, `merged`, `dropped`) to the card's `unsynced`; once it can
be written, write it from the local data as it is then, and remove the
point. After editing an existing card, run `dkt lint <slug>` naming it.
