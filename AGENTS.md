# Threader - Repository Local Agent Guide

# AGENTS.md - System Orchestration & Operational Rules

## 1. Core Operational Directives
1. You are an autonomous software engineering agent tasked with maintaining and developing the **Threader** multi-platform marketing ecosystem.
2. The codebase is written in Go and structured around an extensible **Provider-Plugin pattern** (`internal/platform`) where platforms like Nostr, Threads, and future integrations plug in via a unified driver contract.

---

## 2. Source Control Permissions & Git Policy (STRICT)

### ✅ Automatic Commit and Push
- **Git Operations Permitted**: The agent is permitted and expected to perform Git operations. After implementing any fix, modular feature, or component, the agent must consolidate the modifications, perform a commit with a descriptive message, and push the changes.
- **Do not wait for the user to ask**: Commit + push is part of finishing any self-contained task or modular implementation.
- **Pure Commit Messages (STRICT)**: When committing, NEVER add any co-author metadata (e.g., `Co-authored-by:` headers, names, or emails). Commit messages must contain only the pure, concise commit message description. Leave author identification entirely to the automatic system git configuration.

---

## 3. Architectural Mandates & Platform Extensibility

### 🧩 Provider-Plugin Pattern
- **Platform Isolation**: All social network integrations belong in `internal/platform/<platform_name>/`.
- **Contract Enforcement**: Every driver must implement `platform.PlatformDriver` (`ID()`, `ValidateConfig()`, `Publish()`, `Capabilities()`) and self-register via `platform.Register(...)` in an `init()` block.
- **No Platform Leakage**: Never write platform-specific network requests or signing algorithms inside `internal/synthesis`, `internal/orchestrator`, or `internal/cli`. All dispatching routes through `orchestrator.Dispatcher`.

### ⚡ First-Class Nostr Architecture
- **Cryptographic Security**: Always use Schnorr signatures (secp256k1) and standard NIP-19 Bech32 formats (`nsec`, `npub`).
- **Flexible Keys & Envs**: Private keys must be resolved dynamically from project configs or environment variables (`NOSTR_NSEC`, `NOSTR_NSEC_<PROJECT>`, etc.).

### 🎯 Multi-Platform Targets
- **Target Independence**: Each project namespace (`internal/project`) maintains independent target configurations. Projects can enable or disable Nostr, Threads, or other platforms on demand.

---

## 4. Development & Execution Standards

- **Ota Execution Governance**: `ota.yaml` is the canonical contract for task execution and validation.
  - Run `ota validate` and `ota doctor` to verify contract readiness.
  - Run `ota run test` to verify test suite execution.
  - Run `ota run build` to build binaries.
- **Anyisland & Ota Boundary**: Anyisland is the deployment/orchestration runtime, while Ota is the execution governance layer. Keep their contracts and purpose distinct.
- **Surgical Execution**: Skip unsolicited refactors to unrelated files. Restrict modifications to the exact target scope requested.
- **Zero Speculation**: Fix identified issues cleanly and directly.
