# Convert a change to the docket schema

Use this only for a change that has no step ticked yet, after the user
agrees. Nothing is committed without the user.

1. Work in the change's own worktree.
2. Set `schema: docket` in `openspec/changes/<slug>/.openspec.yaml` first.
3. Write `openspec/changes/<slug>/docket.yaml` with `title`, `kind` and
   `source`, each confirmed with the user, and `base`: the merge base of
   HEAD and the default branch.
4. Run `openspec instructions proposal --change <slug> --json` and add the
   sections the proposal lacks.
5. Give each capability the change adds a `docket.yaml` with a `title`
   beside its delta (`specs/<capability>/docket.yaml`).
6. Write `plan.md` from `tasks.md` with superpowers:writing-plans,
   following `openspec instructions plan --change <slug>`, then delete
   `tasks.md`.
7. Run `dkt lint <slug>` until it reports nothing.

The change then continues at its plan stage: the workflow's plan review
comes next.
