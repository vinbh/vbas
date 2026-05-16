---
name: Wait for user check before pushing milestone-sized commits
description: For M-level milestones (M3+), commit locally but don't push to origin until user has tested and confirmed
type: feedback
---

For milestone commits (anything labeled M0/M1/M2/M3/...), the user wants to test the local binary BEFORE the commit lands on `origin/main`. Polish commits (small UI tweaks, bug fixes, doc edits) can still be commit-and-push as one step.

**Why:** Milestone commits change behavior the user wants to feel for themselves before it's part of the public history. Reverting later is doable (revert commit) but messy. The cost of pausing is one round-trip; the cost of pushing prematurely is a noisy revert plus rework.

**How to apply:**
- For milestone commits (`feat(MN): ...`): `git commit` locally, then **stop and ask** "ready to push, or want to test first?". Provide a concrete test recipe in the same message.
- For polish/fix/docs commits: continue the existing pattern — commit and push together.
- If unsure whether a commit is "milestone-sized": ask, or default to milestone-sized treatment.
- Triggered by 2026-05-02 incident with M3 daemon push: I pushed before user could verify, user wanted to check first.
