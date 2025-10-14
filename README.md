# NOT7 - Not Your Typical Agent

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

**A single-binary declarative agent runtime. No code. No dependencies. Just intelligence.**

---

## Vision

NOT7 is a **config-driven agent runtime** delivered as a **single binary**. Agents are defined as **declarative JSON configurations** and executed with **zero dependencies**. No programming languages. No installations. Just drop the binary, declare your agent in JSON, and run.

---

## Table of Contents

- [Why NOT7?](#why-not7)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
- [Example Agent](#example-agent)
- [Building from Source](#building-from-source)
- [Roadmap](#roadmap)

---

## Why NOT7?

### The Problem with Current Agentic Frameworks

Today's agentic landscape is fragmented and inaccessible. Frameworks like LangGraph and LangChain are powerful but language-dependent - you must write Python or Node.js code. Installation is complex, dependencies break, and despite how fascinating and useful these tools are, non-technical users cannot easily get started. The barrier to entry remains high.

On the other end of the spectrum, no-code platforms (n8n, Zapier, Make) promise simplicity but deliver rigidity. You define workflows through visual interfaces, which seems appealing until you realize you're locked into that vendor's abstractions. Over time, your agents become tightly coupled to proprietary systems. There's no neutral ground - nothing that's both simple AND extensible.

### The Missing Middle Ground

The industry needs something between these extremes. Not too technical and program-dependent on one side. Not too rigid and vendor-locked on the other. A sweet spot that is:

- **Declarative** (like no-code simplicity)
- **Simple** (anyone can read JSON)
- **Extensible** (anyone can build UIs or tooling on top)
- **LLM-compatible** (AI can generate and evolve specs, vs having to generate the whole python / node projects)

### Learning from Integration Platforms

If you look at the integration and ESB space in the past, they got something fundamentally right: transparent XML or JSON-based workflow definitions. You could open the file, read it, understand it, debug it. The definition WAS the documentation.

Current agentic frameworks abandoned this clarity. They've built hidden graph abstractions that become black boxes. Complex internal state machines, opaque routing logic, workflows you can't easily inspect. When something goes wrong, you're debugging code, not reading specifications.

**NOT7 returns to the integration platform philosophy:** transparent, declarative definitions. But we evolve it further. Instead of static workflow definitions, we embrace evolution through versioning. Different JSON versions become your visibility layer. Each version is a complete, inspectable snapshot. Want to know why your agent behaved differently? Compare v3 to v5. Want to debug? Read the JSON. Want to understand evolution? Trace through versions.

**The versioning IS the intelligence. The transparency IS the debuggability.**

This is the evolutionary next stage of integration platforms - applied to agentic systems. Taking what worked (transparent definitions) and adding what's needed (evolutionary optimization).

---

## Key Features

**Config-Driven**  
Single binary with simple configuration file. No programming languages to install.

**Declarative JSON**  
Agents are pure JSON specifications. No code required.

**LLM-Native**  
JSON specs are trivial for LLMs to generate. AI writes specs, NOT7 executes them.

**Transparent Evolution**  
Version your agent specs. See exactly how your agent evolved from v1 to v10.

**Production Ready**  
Server mode with HTTP API and deploy folder watching. Structured logging for observability.

**Zero Dependencies**  
Download one binary and run. Works on macOS, Linux, Windows.

---

## Quick Start

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/not7/core/raw/main/dist/not7-darwin-arm64 -o not7
chmod +x not7
cp not7.conf.example not7.conf
# Edit not7.conf: OPENAI_API_KEY = sk-your-key
./not7 run examples/poem-generator.json
```

**macOS (Intel) / Linux / Windows:**  
See [dist/](dist/) folder for other platform binaries.

**That's it!** The agent executes and generates output.

---

## Example Agent

Simple poem generator:

**`poem-generator.json`:**
```json
{
  "version": "1.0.0",
  "goal": "Generate a poem about AI agents",
  
  "config": {
    "llm": {
      "provider": "openai",
      "model": "gpt-4",
      "temperature": 0.9
    }
  },

  "nodes": [
    {
      "id": "generate_poem",
      "type": "llm",
      "prompt": "Write a creative poem about AI agents...",
      "output_format": "text"
    }
  ],

  "routes": [
    {"from": "start", "to": "generate_poem"},
    {"from": "generate_poem", "to": "end"}
  ]
}
```

**Run:**
```bash
not7 run poem-generator.json
```

**Output:**
```
NOT7 - Agent Runtime
====================

📖 Loading spec: poem-generator.json
✓ Spec loaded successfully

🚀 Starting agent: Generate a poem about AI agents
📋 Version: 1.0.0

⚙️  Executing node: Generate Poem About Agents (llm)
   ✓ Completed in 2850ms (cost: $0.0275)

✅ Execution completed in 2850ms
💰 Total cost: $0.0275

📄 Output:
─────────────────────────────────────
[Your generated poem appears here]
─────────────────────────────────────

💾 Results saved to: poem-generator.json.result.json
```

---

## Building from Source

### Prerequisites

- Go 1.21 or higher
- Make

### Build

```bash
git clone https://github.com/not7/core.git
cd core
cp not7.conf.example not7.conf
# Edit not7.conf with your API key
make build
./not7 run examples/poem-generator.json
```

### Build for All Platforms

```bash
make build-all
```

Creates binaries in `dist/` for macOS, Linux, Windows.

---

## Roadmap

**Tool Integration**  
MCP protocol support for external tools (databases, APIs, file systems)

**Dynamic Routing**  
LLM-driven path selection at runtime for autonomous agent flow

**Scale & Performance**  
Parallel execution and concurrent agent processing

**Provider Ecosystem**  
Support for multiple LLM providers (Anthropic, local models, custom endpoints)

**Production Hardening**  
Security, authentication, rate limiting, monitoring capabilities

---

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contribution

Willing to contribute ? Or Learn ? Ping me - https://www.linkedin.com/in/gnanaguru/