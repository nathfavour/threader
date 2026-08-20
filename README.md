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

## 🚀 Quick Setup

### 1. Installation

```bash
# Clone the repository
git clone https://github.com/nathfavour/threader.git
cd threader

# Build binary
make build
# or using Ota:
ota run build
```

---

## 🛠️ Execution Governance & Tooling

### Ota Setup (Execution Governance)
[Ota](https://ota.run) provides deterministic execution governance and verification:

```bash
# Verify workspace contract and dependencies
ota doctor

# Run tests
ota run test

# Build binary
ota run build
```

### Anyisland Setup (Runtime Deployment)
[Anyisland](anyisland.json) handles isolated container environments and system services:

```bash
# Start or verify Anyisland environment
anyisland up
```

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

## 🔌 Adding New Platform Drivers

To add a new platform (e.g. Bluesky / ATProto):

1. Create `internal/platform/bluesky/driver.go`.
2. Implement `platform.PlatformDriver` (`ID()`, `ValidateConfig()`, `Publish()`, `Capabilities()`).
3. Call `platform.Register(&BlueskyDriver{})` in an `init()` block.

---

## 📄 License

MIT
