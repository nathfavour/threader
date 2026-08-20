# Threader Architecture

Threader is an extensible, autonomous multi-platform publishing and marketing engine designed for decentralized protocols and social networks. While originally built for Threads, the architecture has been modularized to make **Nostr** a first-class design target alongside Threads, Bluesky, and future decentralized platforms via a unified provider-plugin pattern.

---

## 1. Architectural Strategy: Provider-Plugin Pattern

Threader cleanly decouples **orchestration (AI synthesis, OCR/media processing, namespace config, scheduling)** from **platform drivers**.

```text
                         ┌─────────────────────────────┐
                         │   cmd/threader / CLI / App  │
                         └──────────────┬──────────────┘
                                        │
                         ┌──────────────▼──────────────┐
                         │    internal/orchestrator    │
                         │ (Project Context, AI, Media)│
                         └──────────────┬──────────────┘
                                        │
                  ┌─────────────────────┴─────────────────────┐
                  │                                           │
       ┌──────────▼──────────┐                     ┌──────────▼──────────┐
       │   Driver Registry   │                     │  Media Storage Hub  │
       │ (internal/platform) │                     │  (internal/media)   │
       └──────────┬──────────┘                     └──────────┬──────────┘
                  │                                           │
     ┌────────────┼────────────┬─────────────┐     ┌──────────┼──────────┐
     │            │            │             │     │          │          │
┌────▼────┐  ┌────▼────┐  ┌────▼────┐  ┌─────▼─────┤ ┌─▼────────┐ ┌──────▼─┐
│  Nostr  │  │ Threads │  │ Bluesky │  │ Future... │ │ Blossom/ │ │ Imgur/ │
│ Driver  │  │ Driver  │  │ (ATProto│  │           │ │  NIP-96  │ │ S3/CDN │
└─────────┘  └─────────┘  └─────────┘  └───────────┘ └──────────┘ └────────┘
```

---

## 2. Core Pillars

### 1. Extensible Platform Targets (`internal/platform`)
- **Unified Provider Interface**: All platforms implement `PlatformDriver` (`ID()`, `ValidateConfig()`, `Publish()`, `Capabilities()`).
- **Self-Registering Registry**: Platform drivers register in `init()` blocks without touching the core engine.
- **First-Class Nostr Support**: Full NIP-01/NIP-10/NIP-19 support with Schnorr signatures, `nsec`/`npub` Bech32 decoding/encoding, and multi-relay WebSocket pooling.
- **Meta Threads Support**: Container creation, media attachments, pain-point post discovery, and automated reply threads.

### 2. Project Namespaces & Targets (`internal/project`)
- **Multi-Target Configuration**: Each project namespace can independently activate, configure, and route to any combination of targets (e.g. Nostr, Threads, Bluesky).
- **Environment Key Resolution**: Resolves credentials from project settings or environment variables (`NOSTR_NSEC`, `NOSTR_NSEC_<PROJECT>`, `THREADS_TOKEN`, `THREADS_ACCESS_TOKEN`).
- **Brand Context & Manifest**: Dedicated brand voice, audience profiles, and architecture manifests (`README.md`).

### 3. Intelligent Media Engine (`internal/media`)
- **Tesseract OCR**: Extracts text from images and diagrams to build searchable knowledge graphs.
- **Multimodal AI**: Analyzes visual assets for context, aesthetic value, and marketing angles.
- **Transient & Cloud Hosting**: Automated media hosting bridge for platform APIs requiring public URLs.

### 4. Content Synthesis & Orchestrator (`internal/synthesis` & `internal/orchestrator`)
- **AI-Driven Post & Reply Crafting**: Synthesizes human-grade, concise, technical marketing copy tailored to project manifests.
- **Target Dispatcher**: Broadcasts crafted posts across all enabled project targets or specific requested targets.
- **Spine & Biological Metabolism**: Autonomous pulse loops for scheduling, quota management, and continuous engagement.

---

## 3. Directory Layout

```text
threader/
├── cmd/
│   └── threader/             # CLI & daemon entrypoint
├── internal/
│   ├── ai/                   # AI LLM client & completion inference
│   ├── cli/                  # Cobra CLI commands (project, post, queue, status)
│   ├── container/            # Persona / container profile management
│   ├── media/                # OCR extraction & SQLite asset database
│   ├── orchestrator/         # Multi-platform post dispatcher & MarketingCell
│   ├── platform/             # The Extensibility Layer
│   │   ├── driver.go         # Common Driver & Capabilities interface
│   │   ├── registry.go       # Driver registry (Register / Get / List)
│   │   ├── nostr/            # First-class Nostr driver (Schnorr, NIPs, relay pool)
│   │   └── threads/          # Threads driver (Graph API, container workflows)
│   ├── project/              # Project namespace manager & platform targets
│   ├── synthesis/            # AI post synthesis & strict copy validators
│   └── threads/              # Threads client, hosting, and quota management
├── pkg/
│   ├── biology/              # Metabolic activity tracking
│   ├── config/               # Path & environment configuration
│   ├── nostrutil/            # Bech32, nsec/npub utilities, Schnorr key helpers
│   └── spine/                # Cyclic heartbeat and biological pulse runner
├── ota.yaml                  # Ota execution governance contract
└── AGENTS.md                 # Agent directives and development standards
```

---

## 4. How to Add a New Platform Driver

Adding a new social network (e.g. Bluesky / ATProto, Farcaster, or X) requires only 3 steps:

1. Create `internal/platform/<name>/driver.go`.
2. Implement `platform.PlatformDriver` (`ID()`, `ValidateConfig()`, `Publish()`, `Capabilities()`).
3. Call `platform.Register(&MyDriver{})` inside an `init()` block.
