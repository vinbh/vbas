---
name: User git identity
description: Vinayak Bhatt <vinayakbhatt@mit.tc> — git author for commits when global git config isn't set on this machine
type: user
originSessionId: 359993fe-1b21-460a-b941-fe860c086556
---
Full name: **Vinayak Bhatt**
Email: **vinayakbhatt@mit.tc**
GitHub handle: **vinbh** (the local Linux username `vb666` is unrelated; do not assume it's the GitHub handle)

As of 2026-05-01, the user's machine has no global `git config user.name` / `user.email`. Use the identity above via per-command overrides:

```bash
git -c user.name="Vinayak Bhatt" -c user.email="vinayakbhatt@mit.tc" commit ...
```

Do **not** modify global git config without explicit instruction. If a future project uses a different identity (e.g. work email), ask before assuming.
