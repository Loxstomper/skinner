1. Study specs/README.md to learn the application specifications
2. Study the assigned work item in the <work-item> block above
3. Your task is to implement the SINGLE WORK ITEM ONLY per its description using parallel subagents. Before making changes, search the codebase (don't assume not implemented) using Sonnet subagents. You may use up to 500 parallel Sonnet subagents for searches/reads (prefer LSP) and only 1 Sonnet subagent for build/tests. Use Opus subagents when complex reasoning is needed (debugging, architectural decisions).
4. After implementing functionality or resolving problems, run the tests for that unit of code that was improved. If functionality is missing then it's your job to add it as per the application specifications. Ultrathink.
5. When the tests pass, then `git add` the modified files, then `git commit` with a message describing the changes. After the commit, `git push`, close the work item `bd close <ITEM_ID> --reason "complete, git commit: {COMMIT_SHA}"`
99999. Important: When authoring documentation, capture the why — tests and implementation importance.
999999. Important: Single sources of truth, no migrations/adapters. If tests unrelated to your work fail, resolve them as part of the increment.
9999999. You may add extra logging if required to debug issues.
999999999. When you learn something new about how to run the application, update @CLAUDE.md using a subagent but keep it brief. For example if you run commands multiple times before learning the correct command then that file should be updated.
9999999999. For any bugs you notice or spec inconsistencies, resolve them or create a detailed bug work item using Opus 4.6 subagent with 'ultrathink' `bd create "[{NAME}]" --description="{DESCRIPTION}" -t bug -p 1 --deps discovered-from:<ITEM_ID> --json` using a subagent even if it is unrelated to the current piece of work.
99999999999. Implement functionality completely. Placeholders and stubs waste efforts and time redoing the same work.
999999999999. IMPORTANT: Keep @CLAUDE.md operational only. A bloated CLAUDE.md pollutes every future loop's context.
