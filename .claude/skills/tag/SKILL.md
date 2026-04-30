---
description: Create a new semver tag with an annotated message summarizing changes since the last release.
disable-model-invocation: true
allowed-tools:
  - Bash(git log)
  - Bash(git tag)
  - Bash(git describe)
  - Bash(gpg)
---

# Tag workflow

Create a new semver tag based on the changes since the last release tag.

## Steps

1. **Find the latest release tag:**
   ```
   git describe --tags --abbrev=0 --match 'v*'
   ```
   If no tag exists, treat the entire history as new (start from `v0.0.0`).

2. **Review changes since that tag:**
   ```
   git log <last-tag>..HEAD --oneline
   ```
   Read the commit messages and any relevant diffs to understand the nature of changes.

3. **Determine the version bump** by analyzing commits:
   - **patch** (v1.2.X): bug fixes, documentation, internal refactoring, dependency updates
   - **minor** (v1.X.0): new features, new CLI flags, new commands, backward-compatible enhancements
   - **major** (vX.0.0): breaking API/CLI changes, removed commands/flags, incompatible config format changes

4. **Compose a tag message** summarizing the changes. Format:
   ```
   Release vX.Y.Z

   - bullet point per logical change
   - group related commits into single bullets
   ```
   Keep it concise — one line per logical change, not per commit.

5. **Present the plan to the user** before tagging:
   - Proposed version (with reasoning for the bump level)
   - The tag message
   - Ask for explicit confirmation or adjustment

6. **Check for GPG signing capability:**
   ```
   gpg --list-secret-keys 2>/dev/null
   ```
   If a private key is available, use `git tag -s`. Otherwise use `git tag -a`.

7. **Create the tag** (only after user confirms):
   ```
   git tag -s vX.Y.Z -m "<message>"    # if GPG available
   git tag -a vX.Y.Z -m "<message>"    # otherwise
   ```

8. Report the created tag. Do NOT push — let the user decide when to push.
