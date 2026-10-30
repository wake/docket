# Workflow setup

Run setup when `dkt workflow` reports `missing` or `incomplete` at the start
of development work, before that work, or when the user asks to update a
`stale` workflow. It writes the user's workflow file: Docket's template,
with each step said the way this user does it.

1. Run `dkt workflow --json` for the file's path (`result.path`), and
   `dkt workflow template --json` for the template (`result.text`) and its
   hash (`result.template`).
2. Draft the file from the template, step by step, saying for each step
   how this user does it:
   - Look first at the user's instructions already in your context, such
     as their global agent instructions and memory: the reviewers, models,
     commands and habits they name for planning, reviews, verification and
     merging.
   - Then at the tools you can use: other models you can call for a
     review, subagent types, plugins, skills and commands.
   - Do not read, list or search user-level files that are not already in
     your context, such as global instruction files or another agent's
     configuration, unless the user agrees first.
   - Keep every step of the template. Where the user's instructions say
     how to do a step, write it their way, naming the tool and its
     settings; where they add a step, add it; where they contradict a
     step, keep the template's step and note the conflict for the user to
     settle.
   - Keep the headings exactly as the template has them: `## Full review`,
     `## Light review`, `## Check`, `## Direct merge` and `## Reviews`.
   - For `incomplete`, keep the sections the file already has as they
     are and draft only the missing ones; the draft is still the whole
     file. For `stale`, keep the user's tools and bring in what the
     current template does differently.
3. Show the user the whole file as it will be written, in one block with
   every section in full, not a summary or only the new part. Then list
   each point where it departs from the template or from their
   instructions, and the conflicts to settle. Revise it until they agree.
4. Only after the user agrees, write the file at `result.path`, creating
   its directory when missing. If the path is a symbolic link, write
   through it and keep the link. The file starts with the hash from
   step 1:

   ```
   ---
   template: "<hash>"
   ---
   ```

5. Run `dkt workflow`. It must report `ok`; then go on with the work.

If the user does not want to set up now, follow the template's steps for
this piece of work, and ask again at the next one.
