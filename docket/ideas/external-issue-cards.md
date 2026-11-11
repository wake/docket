---
title: Cards for issues kept in another tool
---

`docket/order.md` lists only paths under `docket/ideas/` and `docket/issues/`. When `AGENTS.md` has a `## Issues` section, issues are filed in the tool it names, and the skill says they are not listed in the order: an address written there fails `order-format`, and the whole order is `order-invalid`. Some projects need to put such issues in the order (found in docket-workbench, 2026-11-11).

Listing addresses in the order was weighed and set aside: dkt cannot get their titles or check that they exist, and work writes the same address in `--source` in more than one way.

## Direction (maintainer, 2026-11-11)

A thin card in `docket/issues/`: front matter with `title`, `source` and `external: <address>`, while the content stays in the other tool. The order lists the card's path, work that handles it writes the card's path in `--source`, and whether it is taken follows the rule for local issues. The order rules and the `order-format` and `order-item` checks stay as they are, `result.next` has a title, and viewers list the card with the other issues.

A card can be made by hand today, but `dkt lint` asks every issue the branch adds for all four sections, so each needs a line, such as one pointing to the address, to pass.

What Docket adds:

- The skill and the specs know cards, and the sentences saying that an issue in another tool is opened there, is named by its address in `--source` and is not listed in the order change to match.
- `dkt issue <name> --external <address>` writes a card.
- `dkt lint` checks `external`, and `dkt status` counts an item as taken when a `source` is the card's path or its `external` value.

Out of scope: keeping the title in step with the other tool, and noticing that the issue was closed there.

## Open

- When to make a card: only for an issue that goes into the order (suggested, so that the other tool and `docket/issues/` do not become two lists to keep in step), or for every issue filed there.
- A card's checks: with `external`, no section is required and the body may be empty (suggested), or one section is required.
- The form of `external`: one line, not empty, as for `source` (suggested, since some tools use numbers rather than URLs), or a URL.
- Taken by the `external` value: the match is exact, so a trailing `/` or a `#` part makes it miss. New work writes the card's path; the address match is there for work started before the card existed.
- Closing by hand when the issue is closed in the other tool without a change: `dropped` is ignored on issues today, and deleting a card the order still lists makes the order `order-invalid`. Suggested: let a card, or any issue, take `dropped: true`, which `result.next` skips.
