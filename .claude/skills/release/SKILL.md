---
description: Create a gogogo release — pre-release if HEAD has no tag, stable release if it does.
disable-model-invocation: true
allowed-tools:
  - Bash(go build)
  - Bash(go test)
  - Bash(gogogo release)
  - Bash(git tag)
  - Bash(git describe)
  - Bash(git push)
---

# Release workflow

Determine the release type and run `gogogo release`.

## Steps

1. Check if the current commit has a semver tag:
   ```
   git describe --exact-match --tags HEAD 2>/dev/null
   ```

2. **If a tag exists** (e.g. `v1.2.3`): ensure the tag is pushed to the remote, then run a stable release:
   ```
   git push origin <tag>
   gogogo release
   ```

3. **If no tag exists**: ensure local commits are pushed, then create a pre-release:
   ```
   git push origin HEAD
   gogogo release --commit=@
   ```

4. Before releasing, ensure the code compiles and tests pass:
   ```
   go build ./... && go test ./...
   ```
   If either fails, stop and report the error — do not release broken code.

5. Report the result: tag used, whether it was a stable or pre-release, and any errors.
