# 🛡️ git-attest

**Know the intent behind every pull request.**

A transparency layer for the age of AI-assisted code. `git-attest` silently captures *how* code was written — the prompts, the iteration patterns, the file modification forensics — and attaches it as an immutable certificate to every commit.

It doesn't detect malicious code. It doesn't judge *what* the code does. It surfaces the signals that let a reviewer quickly understand whether a contribution was **intentful** — a developer who prompted thoughtfully, iterated on their work, and reviewed before committing — or **low-effort** — code that appeared in a single batch with no visible human involvement. Regardless of what tools were used to write it.

---

## The Problem

Open source maintainers are facing a new reality:

- **AI agents are submitting pull requests** — sometimes autonomously, sometimes with a human in the loop, sometimes with no oversight at all.
- **Traditional code review doesn't scale** when a single contributor can generate hundreds of files in minutes.
- **You can't tell intent from a diff.** A perfectly formatted PR could be a thoughtful contribution or an automated supply chain attack. The code looks the same.

What's missing isn't better static analysis. It's **provenance** — evidence of *how* the code was constructed, not just what it contains.

## What git-attest Does

`git-attest` captures two categories of evidence on every commit:

### 🎙️ Active Telemetry — Proof of Intent
Extracts the developer's prompts from their AI coding tools:

| Tool | What's Captured |
|------|----------------|
| Claude Code | User prompts from `~/.claude/projects/` session logs |
| Antigravity (Gemini) | User prompts from `transcript.jsonl` conversation logs |
| GitHub Copilot | Chat prompts from VS Code workspace state |
| Cursor | Chat prompts from Cursor workspace state |

If a developer asked their AI assistant *"Add an authentication endpoint with rate limiting"*, that prompt is captured. If there are no prompts — the code appeared without any recorded human intent — that absence is itself a signal.

### 🔬 Passive Forensics — Proof of Construction
Analyzes the filesystem metadata of every staged file:

| Metric | What It Reveals |
|--------|----------------|
| `avg_mod_interval_ms` | Average time between file edits (human pacing vs machine bursting) |
| `max_mod_interval_ms` | Longest pause between edits (deep thinking vs constant generation) |
| `min_mod_interval_ms` | Shortest time between edits (detects superhuman file writes) |
| `ctime_mtime_drift` | Were file timestamps tampered with after creation? |
| `first_to_last_file_mod_gap_ms` | Total time span of the coding session |
| `last_mod_to_commit_gap_ms` | Did the developer pause to review before committing? |
| `newly_created_file_count` | How many files are brand new vs. modified? |

### 🔒 Privacy by Default
All captured prompts are automatically scrubbed for:
- AWS access keys and secrets
- JWT tokens
- IP addresses
- API keys and tokens
- Custom patterns via a `.tracefilter` file

The evidence never leaves your Git infrastructure. It travels as a Git Note — stored on your own GitHub/GitLab server, not a third-party service.

---

## Reading the Signals — What Intent Looks Like

Different workflows leave different fingerprints. Here's how three real-world scenarios would appear in a git-attest certificate:

### Scenario 1: Developer using an Agentic IDE (Cursor, Antigravity)

A developer uses an agentic IDE to build a feature. They prompt iteratively, review the output, and commit.

```
Prompts:              ██████████  12 captured
Pacing (Min/Avg/Max): 8s / 15s / 45s      Files edited with human-like pauses between them
Review gap:           45,000ms    45 seconds between last edit and commit
ctime drift:          0           No timestamp tampering
```

**What this tells the reviewer:** Someone was in the loop. They gave 12 distinct instructions, the edit pacing was slow (not superhumanly fast), and they paused to review before committing. This is an intentful contribution — the AI was a tool, not the driver.

---

### Scenario 2: Developer using Claude Code (terminal agent)

A developer uses Claude Code to implement a complex refactor. The prompts are captured from Claude's session logs.

```
Prompts:              ██████      6 captured (from ~/.claude/projects/)
Pacing (Min/Avg/Max): 2s / 4s / 12s       Faster generation, but distinct pauses between files
Review gap:           120,000ms   2 minutes between last edit and commit
ctime drift:          0           No timestamp tampering
```

**What this tells the reviewer:** Clear human-in-the-loop pattern. Fewer prompts but a longer review gap suggests the developer read through the changes carefully before committing. Intentful.

---

### Scenario 3a: Intentful developer using a background agent

A developer uses a background agent (OpenClaw, Devin, etc.) to build a feature. They iteratively prompt the agent, review its output, and prompt it again. Even though we can't capture the prompts, the file timestamps show bursts of machine activity separated by human deliberation.

```
Prompts:              ░░░░░░░░░░  0 captured (no session logs found)
Pacing (Min/Avg/Max): 15ms / 2m / 8m      Agent bursts separated by long human review pauses
Review gap:           300,000ms   5 minutes between last file write and commit
ctime drift:          0           No timestamp tampering
```

**What this tells the reviewer:** No prompts — we can't see the initial instruction. However, the pacing shows a clear pattern: fast file writes (agent execution) separated by multi-minute pauses (human review and reprompting). The human was in the loop steering the agent, even if we couldn't record their keystrokes. Intentful, just a different workflow.

---

### Scenario 3b: Lazy developer using a background agent

Same tool, but the developer lets the agent commit directly without reviewing the output.

```
Prompts:              ░░░░░░░░░░  0 captured (no session logs found)
Pacing (Min/Avg/Max): 2ms / 15ms / 50ms   Files generated in a superhuman burst
Review gap:           200ms       Near-instant commit after last file write
ctime drift:          0           No timestamp tampering
```

**What this tells the reviewer:** No prompts AND a near-instant commit. Nobody paused to read the output. The code went from agent to commit to PR with no visible human checkpoint. This doesn't mean the code is bad — but there's **no evidence anyone looked at it** before it landed in your review queue.

---

### Scenario 4: Suspicious contribution

Someone submits a large PR with fabricated or absent provenance.

```
Prompts:              ░░░░░░░░░░  0 captured
Pacing (Min/Avg/Max): 0ms / 0ms / 0ms     All 47 files share the exact same millisecond timestamp
Review gap:           50ms        Commit was essentially instant
ctime drift:          3           3 files have ctime ≠ mtime (timestamps were touched)
```

**What this tells the reviewer:** Zero prompts, zero pacing interval, instant commit, and evidence of timestamp manipulation. Every signal points to code that was bulk-generated or copied from elsewhere with no human review. This PR deserves significantly more scrutiny.

---

> **The key insight:** git-attest doesn't decide if code is good or bad. It surfaces a consistent set of signals — prompts, pacing, review gaps, timestamp integrity — and lets the reviewer make an informed judgment. An intentful developer using *any* tool will naturally produce a different fingerprint than an unattended process.

---

## Install

```bash
brew install git-attest
```

That's it. No per-repo setup. No configuration files. Every repository on your machine is automatically instrumented.

<details>
<summary><strong>Other installation methods</strong></summary>

**Go Install:**
```bash
go install github.com/SaaSAILabs/attest-cli@latest
git attest init   # one-time global setup
```

**From Source:**
```bash
git clone https://github.com/SaaSAILabs/attest-cli.git
cd attest-cli
go build -o git-attest main.go
sudo mv git-attest /usr/local/bin/
git attest init
```
</details>

---

## Usage

### You don't change your workflow.

After installation, just commit and push like you always have:

```bash
# Code with your AI tool of choice...

git add .
git commit -m "Add payment endpoint"
# → [attest] ✓ flight recorder attached (12 events)

git push
# → Transparency certificate pushed alongside your code
```

For explicit branch pushes, use `git attest push` to ensure the certificate travels with your code:

```bash
git attest push origin feature-branch
```

### Preview a certificate before committing

```bash
git attest preview
```

### Disable temporarily

```bash
ATTEST_DEV_MODE=1 git commit -m "WIP local stuff"
```

### Uninstall cleanly

```bash
git attest uninstall
```

---

## What a Transparency Certificate Looks Like

Every commit gets a JSON payload attached as a Git Note. Here's an annotated example:

```jsonc
{
  // Schema version for forward compatibility
  "version": "0.1.0",

  // Which AI tools were active during this commit
  "profile": "antigravity",

  // When the commit was made (ms since epoch)
  "commit_timestamp": 1781133610933,

  // High-level forensic summary
  "summary": {
    // Time between first and last file modification (ms)
    // A 60-second spread suggests iterative work, not a single dump
    "first_to_last_file_mod_gap_ms": 60246,

    // Time between last file save and commit (ms)
    // A 40-second gap suggests the developer paused to review
    "last_mod_to_commit_gap_ms": 39771,

    // Total number of recorded events (prompts + file mods)
    "total_prompt_events": 5
  },

  // Filesystem-level forensic analysis
  "forensics": {
    // Average time between file modifications (ms)
    // 15,000ms = human pacing; 10ms = machine bursting
    "avg_mod_interval_ms": 15000,

    // Longest pause between file modifications
    "max_mod_interval_ms": 120000,

    // Shortest pause between file modifications
    "min_mod_interval_ms": 8000,

    // Number of files where ctime ≠ mtime (potential timestamp tampering)
    "files_with_ctime_drift": 0,

    // Maximum ctime-mtime drift across all files (ms)
    "max_ctime_mtime_drift_ms": 0,

    // How many staged files were created during this session
    "newly_created_file_count": 1,

    // Total files in this commit
    "total_staged_files": 3
  },

  // Chronological event log
  "flight_recorder": [
    {
      "timestamp": 1781133308000,
      "type": "agent_prompt",
      "meta": {
        "prompt": "Can we make this part of the attest cli build itself?",
        "source": "antigravity"
      }
    },
    {
      "timestamp": 1781133510914,
      "type": "file_modification",
      "meta": {
        "file": "cmd/install.go",
        "mtime": 1781133510914,
        "ctime": 1781133510914,
        "btime": 1780709362398
      }
    }
    // ... more events
  ]
}
```

---

## How It Works

```
Developer's Machine                          GitHub
┌─────────────────────────────────────┐     ┌──────────────────┐
│                                     │     │                  │
│  IDE (Claude / Cursor / Copilot)    │     │  Pull Request    │
│         │                           │     │       │          │
│         ▼                           │     │       ▼          │
│  Session Logs (prompts, tools)      │     │  Commit + Note   │
│         │                           │     │  ┌────────────┐  │
│         │  ┌──────────────────┐     │     │  │ Code Diff  │  │
│         └─▶│   git-attest     │     │     │  ├────────────┤  │
│            │                  │     │     │  │ Flight     │  │
│  Files ──▶ │ prepare-commit-  │────────────▶ │ Recording  │  │
│  (mtime,   │ msg hook         │     │     │  │ (Git Note) │  │
│  ctime,    │                  │     │     │  └────────────┘  │
│  btime)    │ ┌──────────────┐ │     │     │                  │
│            │ │Privacy Filter│ │     │     └──────────────────┘
│            │ └──────────────┘ │     │
│            └──────────────────┘     │
└─────────────────────────────────────┘
```

1. **On commit:** The `prepare-commit-msg` hook triggers `git-attest internal-hook`
2. **Harvest prompts:** Extracts user prompts from all detected AI tool session logs
3. **Collect forensics:** Reads `mtime`, `ctime`, and `btime` of every staged file
4. **Scrub secrets:** Runs all prompts through the privacy redactor
5. **Attach:** Serializes the payload as JSON and attaches it via `git notes add`
6. **On push:** Git Notes travel to the remote alongside your code

---

## Threat Model — What This Is and Isn't

> **Honesty is a feature.**

### ✅ What git-attest provides
- **Transparency, not security.** It makes the construction process *visible*, not *verified*.
- **A trust signal, not a verdict.** Reviewers get evidence to inform their judgment.
- **An anti-tampering anchor.** If prompts exist, the forensic timestamps should corroborate them. Mismatches are suspicious.

### ⚠️ What git-attest does NOT do
- **Detect malicious code.** It captures *how* code was made, not *what* it does.
- **Prove authorship.** It cannot definitively tell you if a human or AI wrote a specific line.
- **Prevent forgery.** A sophisticated attacker could fabricate session logs. The forensic cross-correlation makes this harder, but not impossible.
- **Guarantee completeness.** If a tool doesn't leave session logs (e.g., a web-based AI chat), there will be no prompts to capture.

### 🔮 Future hardening
- Cryptographic signing of payloads (integration with Sigstore/Fulcio)
- Server-side cross-validation via GitHub Actions

---

## Roadmap

- [x] Core evidence capture (prompts + forensics)
- [x] Privacy redaction pipeline
- [x] Claude Code, Antigravity, Copilot, Cursor extractors
- [x] Zero-config global installation
- [x] `git attest push` for explicit branch pushes
- [ ] GitHub Action for PR-level analysis (Intent Card)
- [ ] Terminal command history capture (proof that tests were run)
- [ ] npm distribution (`npx git-attest`)
- [ ] Cryptographic payload signing

---

## Contributing

Contributions are welcome. Please open an issue first to discuss what you'd like to change.

```bash
git clone https://github.com/SaaSAILabs/attest-cli.git
cd attest-cli
go test ./...
go build -o git-attest main.go
```

---

## License

[Apache 2.0](LICENSE) — SaaS AI Labs, Inc.