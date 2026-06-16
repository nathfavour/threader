# Threader Architecture

Threader is an autonomous agentic system designed for automated product marketing on Meta's Threads platform. It operates as a specialized agent within the Vibe Auracle ecosystem, focusing on high-quality, human-like content generation and lead acquisition.

## Core Pillars

### 1. Project Namespaces
- **Isolation**: Each product or brand is managed within a dedicated namespace.
- **Context**: Every namespace maintains its own brand voice, target audience profiles, and historical data.
- **Registry**: A central registry for managing active marketing projects.

### 2. Intelligent Media Engine
- **Storage**: Infinite media upload capability per project.
- **Indexing**: 
  - **Tesseract (OCR)**: Extracts text from images and documents to build a searchable knowledge base.
  - **Multimodal AI**: Analyzes visual content to understand context, aesthetics, and marketing potential.
- **Search**: Semantic search over indexed media to find the best assets for a post.

### 3. Content Synthesis
- **AI-Driven Post Crafting**: Generates human-sounding Threads posts tailored to the product's brand voice.
- **Context Awareness**: Leverages indexed media and project context to ensure relevance and quality.
- **Strategy**: Automatically determines the best time to post and the best content mix (text, image, carousel, video).

### 4. Threads API Integration
- **Mastery**: Full utilization of the Threads API for publishing, analytics, and engagement.
- **Automation**: Handles OAuth flows, container creation, and publication status tracking.
- **Analytics**: Monitors post performance and adjusts strategy accordingly.

## Directory Structure

- `cmd/threader/`: Entry point for the agent daemon.
- `internal/project/`: Namespace and project configuration management.
- `internal/media/`: Media storage, Tesseract OCR, and AI indexing logic.
- `internal/threads/`: Threads API client and publishing orchestrator.
- `internal/ai/`: Integration with Vibe Auracle for LLM/Multimodal capabilities.
- `pkg/`: Reusable utilities for the agent ecosystem.

## Integration with Vibe Auracle

Threader acts as a "Social Agent" that plugs into the **Vibe Auracle** provider.
- **AI Capabilities**: Offloads heavy inference and LLM tasks to the vibe-brain.
- **Vault**: Retrieves Threads API credentials securely via Vibe Vault.
- **Context**: Syncs high-level project goals and learnings with the global vibe-context.
