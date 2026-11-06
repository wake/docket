# Convert a change to the docket schema

Use this only for a change that has no step ticked yet, after the user
agrees. Nothing is committed without the user.

1. Work where "Where to work" in `docket/workflow.md` says.
2. First set `schema: docket` in `openspec/changes/<slug>/.openspec.yaml`.
3. Add front matter at the top of `proposal.md`: `title`, `kind` and
   `source`, each confirmed with the user, and a name under `specs:` for
   each capability the change adds.
4. Run `openspec instructions proposal --change <slug> --json` and add the
   sections the proposal lacks.
5. Write `plan.md` from `tasks.md` with superpowers:writing-plans,
   following `openspec instructions plan --change <slug>`, then delete
   `tasks.md`.
6. Run `dkt lint <slug>` until it reports nothing.

The change then continues at its plan stage: next, follow "Plan review"
in `docket/workflow.md`.
