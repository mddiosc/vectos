## 1. Spec And Setup Contract

- [x] 1.1 Extend the `agent-setup` change artifacts to cover optional global guidance for supported agents
- [x] 1.2 Document the OpenCode-specific guidance contract and fallback order

## 2. OpenCode Guidance Management

- [x] 2.1 Add managed global guidance handling for `vectos setup opencode`
- [x] 2.2 Preserve existing user guidance and require confirmation before appending to an existing global file
- [x] 2.3 Ensure reruns update only the managed guidance block without duplication

## 3. Validation

- [x] 3.1 Verify the setup flow when no global OpenCode guidance file exists
- [x] 3.2 Verify the setup flow when an existing global OpenCode guidance file is already present
- [x] 3.3 Update README/setup documentation to reflect the Vectos-first retrieval behavior
