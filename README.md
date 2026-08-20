# Threader 🧵

Threader is an autonomous, multi-platform marketing and engagement engine. It combines AI copy synthesis, Tesseract OCR media indexing, and a modular **Provider-Plugin architecture** to automate marketing across **Nostr**, **Meta's Threads**, and any future decentralized social platforms.

---

## 🌟 Key Capabilities

- **Modular Platform Drivers**: Built around a unified `PlatformDriver` contract. Nostr and Threads are plug-and-play targets.
- **First-Class Nostr Support**: NIP-01 Kind 1 text notes, NIP-10 reply threading, NIP-19 `nsec`/`npub` Bech32 key support, Schnorr event signing, and multi-relay WebSocket pooling.
- **Multi-Target Project Namespaces**: Configure multiple projects/personas, each with granular control over enabled publishing targets (e.g. publish to Nostr only, Threads only, or both simultaneously).
- **Environment Key Resolution**: Store keys directly in project configs or reference custom environment variables (`NOSTR_NSEC`, `NOSTR_NSEC_<PROJECT>`, `THREADS_TOKEN`, etc.).
- **Intelligent Media Engine**: Automatically indexes screenshots and diagrams using Tesseract OCR and multimodal AI context extraction.
- **AI Content Synthesis**: Generates dense, high-signal, human-sounding marketing copy tailored strictly to your project's architecture manifest.

---

## 📦 Installation

Threader is distributed and managed for end users via [Anyisland](https://github.com/nathfavour/anyisland), a decentralized, platform-agnostic package manager that resolves runtime system dependencies and builds the application binary automatically.

### 1. Install Anyisland

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/nathfavour/anyisland/master/install.sh | bash
```

### 2. Initialize Island Environment

```bash
anyisland setup
```

### 3. Ingest and Install Threader

```bash
anyisland ingest github.com/nathfavour/threader
```

*Direct script install fallback:*
```bash
curl -fsSL https://raw.githubusercontent.com/nathfavour/threader/master/install.sh | bash
```

---

## 🛠️ Contributing & Development

Threader's development lifecycle, verification gates, and agent safety policies are governed deterministically by [Ota](https://ota.run).

### 1. Clone the Repository

```bash
git clone https://github.com/nathfavour/threader.git
cd threader
```

### 2. Install Ota

```bash
# macOS / Linux
curl -fsSL https://dist.ota.run/install.sh | sh

# Windows PowerShell
irm https://dist.ota.run/install.ps1 | iex
```

### 3. Verify Workspace Readiness

Inspect system dependencies, toolchains, and agent safety boundaries:

```bash
ota doctor
```

### 4. Run Test Suite & Build

Execute the test suite and binary builds governed by `ota.yaml`:

```bash
# Run unit and integration tests
ota run test

# Build binary
ota run build
```

### 5. Adding New Platform Drivers

Adding support for any new social platform (e.g. Bluesky / ATProto, Farcaster) requires only 3 steps:

1. Create `internal/platform/<name>/driver.go`.
2. Implement `platform.PlatformDriver` (`ID()`, `ValidateConfig()`, `Publish()`, `Capabilities()`).
3. Call `platform.Register(&MyDriver{})` inside an `init()` block.

---

## 📋 CLI Usage

### 1. Managing Projects & Platform Targets

```bash
# Create a new project with Nostr nsec and Threads token
threader project create "MyProduct" \
  --desc "Decentralized protocol" \
  --nsec "nsec1..." \
  --threads-token "TH_..." \
  --relays "wss://relay.damus.io,wss://nos.lol"

# List all projects and their active targets
threader project list

# Interactively edit project settings and target credentials
threader project edit [project-id]

# Toggle publishing targets for a project
threader project target [project-id] enable nostr
threader project target [project-id] disable threads
```

### 2. Crafting and Publishing Posts

```bash
# Craft an AI post grounded in the project manifest
threader post craft -p "MyProduct"

# Craft and attach media
threader post craft -p "MyProduct" --media ./screenshot.png

# Publish directly to all enabled targets (Nostr + Threads)
threader post publish "Building decentralized tools." -p "MyProduct"

# Publish specifically to Nostr only
threader post publish "Nostr-only announcement." -p "MyProduct" -t nostr

# Publish specifically to Threads only
threader post publish "Threads-only update." -p "MyProduct" -t threads
```

### 3. Media Queue & Autonomous Daemon

```bash
# Add media to the automated queue
threader queue add ./diagram.png -p "MyProduct"

# Start the background daemon
threader start

# Check status of daemon and targets
threader status

# View real-time daemon logs
threader logs
```

---

## 📄 License

MIT
