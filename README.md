# OpenCodeForge

A local, Docker-based AI coding workspace. Think of it as a self-hosted, fully
offline alternative to a hosted coding assistant: an LLM server, a chat UI, a
sandboxed tools API that can read/write a single workspace directory, and an
optional repository indexer.

OpenCodeForge is designed for a developer with **47–64 GB RAM** on Windows
(Docker Desktop / WSL2), Linux, or macOS, and works on CPU-only or with an
NVIDIA GPU.

---

## Architecture

```
                +--------------------------+
                |   Browser / VS Code      |
                +-------------+------------+
                              |
                              v
                  +-----------+-----------+
                  |  Open WebUI  :3000    |   chat UI, prompts library
                  +-----------+-----------+
                              |
                              v
                  +-----------+-----------+
                  |   Ollama     :11434   |   model server
                  +-----------+-----------+
                              |
        +---------------------+---------------------+
        |                     |                     |
        v                     v                     v
 qwen2.5-coder:14b   qwen2.5-coder:32b   deepseek-coder-v2:lite

                  +-----------------------+
                  |  Tools API   :8088    |   safe file/git/shell access
                  +-----------+-----------+
                              |
                              v
                       ./workspace        <- your target repo lives here

                  +-----------------------+
                  |  Indexer  (profile)   |   walks workspace -> JSONL
                  +-----------------------+
```

The LLM never touches the host directly. It calls the **Tools API**, which
sandboxes every operation inside `./workspace` and only runs commands from a
configurable allowlist.

---

## Components

| Service       | Port  | Purpose                                                   |
|---------------|-------|-----------------------------------------------------------|
| `ollama`      | 11434 | Local model server (Ollama). Loads coding models.         |
| `open-webui`  | 3000  | Web chat UI, talks to Ollama via `OLLAMA_BASE_URL`.       |
| `tools-api`   | 8088  | Go HTTP API: list/read/write/patch/search/git/run.        |
| `indexer`     | -     | Optional Python service. Profile `tools`. Builds JSONL.   |

---

## Requirements

- Docker 24+ with the Compose v2 plugin (`docker compose ...`)
- 47–64 GB RAM recommended (16 GB will only fit `:7b`)
- ~50 GB free disk for models
- Optional: NVIDIA GPU + recent drivers + NVIDIA Container Toolkit

---

## Quick start

```bash
git clone https://github.com/coinbaseblock/OpenCodeForge.git opencodeforge
cd opencodeforge
cp .env.example .env

# Bring up Ollama, Open WebUI, and the tools API.
docker compose up -d --build

# Pull a coding model (recommended default).
docker exec -it opencodeforge-ollama ollama pull qwen2.5-coder:14b
```

Open the chat UI at <http://localhost:3000>.

If you prefer Make:

```bash
make up
make pull-default
make webui-url
```

---

## Recommended models

| Model                      | RAM (CPU q4) | When to use                                  |
|----------------------------|--------------|----------------------------------------------|
| `qwen2.5-coder:7b`         | ~6 GB        | Autocomplete, small scripts, fastest replies |
| `qwen2.5-coder:14b`        | ~10–12 GB    | Default for serious coding & refactors       |
| `qwen2.5-coder:32b`        | ~24–28 GB    | Long reasoning, large repos, best quality    |
| `deepseek-coder-v2:lite`   | ~10–14 GB    | MoE alternative, strong on code reasoning    |

CPU-only? Start with `:14b`. `:32b` works on 64 GB RAM but is slow without a
GPU.

Pull them with:

```bash
make pull-default     # 14b
make pull-light       # 7b
make pull-heavy       # 32b
make pull-deepseek    # deepseek-coder-v2:lite
make pull-all         # everything above
```

Or use the helper scripts under `scripts/`.

---

## Using your own repo

Drop or clone a project into `./workspace`:

```bash
git clone https://github.com/your/project workspace/project
```

Everything below `./workspace` is mounted into:

- `tools-api` at `/workspace` (read/write, sandboxed)
- `indexer` at `/workspace` (read-only)

The Tools API refuses any path that escapes `/workspace`.

### Build an index (optional)

```bash
make index
```

This produces `./data/index/repo_index.jsonl` with one JSON record per chunk
(`path`, `lang`, `size`, `start_line`, `end_line`, `text`). You can feed those
chunks into a RAG step or inspect them manually.

---

## Using Open WebUI

1. Open <http://localhost:3000>.
2. The first time, create a local account if `WEBUI_AUTH=true`. With the
   default `WEBUI_AUTH=false` you go straight in.
3. Pick a model (top-left). The pulled coder models will appear automatically.
4. Open **Workspace → Prompts** and import the markdown files from `./prompts`
   (or paste them as system prompts per chat).

The most useful prompts are:

- `prompts/system-coder.md` – default behavior for the assistant
- `prompts/repo-analyst.md` – ask it to map an unknown repo
- `prompts/patch-writer.md` – force unified-diff output for safer edits
- `prompts/docker-debugger.md` – step-by-step Compose debugging
- `prompts/go-expert.md` – idiomatic Go specialist

---

## Calling the Tools API

Health check:

```bash
curl http://localhost:8088/health
```

List files (relative to `/workspace`):

```bash
curl 'http://localhost:8088/files?path=.'
```

Read a file:

```bash
curl 'http://localhost:8088/read?path=project/main.go'
```

Write a file (sandboxed):

```bash
curl -X POST http://localhost:8088/write \
  -H 'Content-Type: application/json' \
  -d '{"path":"project/notes.md","content":"# Hello\n"}'
```

Apply a unified diff:

```bash
curl -X POST http://localhost:8088/patch \
  -H 'Content-Type: application/json' \
  -d '{"diff":"--- a/project/main.go\n+++ b/project/main.go\n@@ ...\n"}'
```

Search:

```bash
curl 'http://localhost:8088/search?q=TODO'
```

Git status / diff (runs inside the workspace):

```bash
curl 'http://localhost:8088/git/status?path=project'
curl 'http://localhost:8088/git/diff?path=project'
```

Run an allowlisted command:

```bash
curl -X POST http://localhost:8088/run \
  -H 'Content-Type: application/json' \
  -d '{"cmd":"go test ./...","cwd":"project"}'
```

The full schema is in [`tools-api/README` of the source](./tools-api).

---

## Safety model

OpenCodeForge assumes the LLM may make mistakes, so the Tools API is the only
path through which it touches your filesystem.

- **Path sandbox.** Every request is resolved against `/workspace`. Paths
  containing `..`, absolute host paths, or symlinks that escape the sandbox
  are rejected.
- **Command allowlist.** `POST /run` only accepts commands whose full
  command line begins with one of the entries in `ALLOWED_COMMANDS`
  (defaults: `go test ./...`, `go build ./...`, `npm test`, `npm run build`,
  `python -m pytest`, `docker compose config`).
- **Hard deny list.** Patterns like `rm -rf /`, `format`, `del /s`,
  `shutdown`, `powershell -EncodedCommand`, and pipe-to-shell installers
  (`curl ... | sh`, `iwr ... | iex`) are rejected even when `SAFE_MODE` is
  off.
- **`SAFE_MODE=true`** is the default. Setting it to `false` only relaxes the
  allowlist – the deny list and path sandbox always apply.
- **No secrets.** Don't put `.env` files with production secrets into
  `./workspace`. The Tools API has no notion of secret redaction.

---

## NVIDIA GPU (optional)

If you're on Linux or WSL2 with an NVIDIA GPU and have the [NVIDIA Container
Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)
installed, append this to the `ollama` service in `docker-compose.yml`:

```yaml
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
```

Then `docker compose up -d --force-recreate ollama`.

---

## Troubleshooting

| Symptom                                       | Try                                                      |
|-----------------------------------------------|----------------------------------------------------------|
| Open WebUI shows no models                    | `make models`; if empty, `make pull-default`             |
| `ollama` keeps OOM-killing                    | Use a smaller model or set `OLLAMA_KEEP_ALIVE=0`         |
| `tools-api` returns 403 on a path             | The path resolved outside `/workspace`; check symlinks   |
| `POST /run` returns `command not allowed`     | Add the command prefix to `ALLOWED_COMMANDS` in `.env`   |
| Patches fail to apply                         | Ensure your diff is a real unified diff with `a/` `b/`   |
| Port already in use                           | Change `OLLAMA_PORT`/`WEBUI_PORT`/`TOOLS_PORT` in `.env` |
| Windows: `make` not found                     | Use the PowerShell scripts in `scripts/` instead         |

---

## Layout

```
opencodeforge-local-ai/
├─ docker-compose.yml
├─ .env.example
├─ Makefile
├─ ollama/Modelfile.coder
├─ tools-api/         # Go HTTP API (sandboxed file/git/shell)
├─ indexer/           # Python repo indexer (JSONL output)
├─ prompts/           # System prompts for the assistant
├─ workspace/         # YOUR target repo goes here
└─ scripts/           # pull-models / healthcheck / reset
```
