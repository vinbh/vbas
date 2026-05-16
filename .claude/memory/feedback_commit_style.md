---
name: Commit message style — no Claude trailer
description: Do not append "Co-Authored-By: Claude" trailer to commit messages
type: feedback
originSessionId: 359993fe-1b21-460a-b941-fe860c086556
---
User does not want the `Co-Authored-By: Claude Opus X <noreply@anthropic.com>` trailer that the default Claude Code commit flow appends. Author commits solely under their git identity.

**Why:** Asked explicitly during the first commit on the vbas project (2026-05-01). User wants clean commit history under their own name only.

**How to apply:** Use the standard commit flow (HEREDOC for message, etc.) but stop short of the Co-Authored-By line. Applies project-wide and likely to all future projects with this user — extend the same default elsewhere unless they signal otherwise.
