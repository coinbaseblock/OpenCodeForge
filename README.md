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
| `code-server` | 8443  | Browser-based VS Code, same `./workspace` mount.          |
| `indexer`     | -     | Optional Python service. Profile `tools`. Builds JSONL.   |

---

## Requirements

- Docker 24+ with the Compose v2 plugin (`docker compose ...`)
- 47–64 GB RAM recommended (16 GB will only fit `:7b`)
- ~50 GB free disk for models
- Optional: NVIDIA GPU + recent drivers + NVIDIA Container Toolkit

---

## Quick start

End-to-end, from clone to chatting in Open WebUI:

```bash
# 1. Get the code
git clone https://github.com/coinbaseblock/OpenCodeForge.git opencodeforge
cd opencodeforge

# 2. Configure
cp .env.example .env

# 3. Start Ollama, Open WebUI, and the tools API
docker compose up -d --build

# 4. Pull a model profile (skips models you already have)
./scripts/pull-models.sh default        # qwen2.5-coder:14b

# 5. Verify
./scripts/healthcheck.sh
open http://localhost:3000              # macOS / xdg-open / browser
```

On **Windows 11 (PowerShell)** without `make`:

```powershell
copy .env.example .env
docker compose up -d --build
powershell -ExecutionPolicy Bypass -File .\scripts\pull-models.ps1 default
.\scripts\healthcheck.ps1
start http://localhost:3000
```

If you prefer Make:

```bash
make up
make pull-default
make health
make webui-url
```

---

## Recommended models

| Model                      | RAM (CPU q4) | When to use                                  |
|----------------------------|--------------|----------------------------------------------|
| `qwen2.5-coder:1.5b`       | ~2 GB        | Smoke test, tiny boxes, snippet completion   |
| `qwen2.5-coder:3b`         | ~3–4 GB      | Fast autocomplete, low-latency edits         |
| `qwen2.5-coder:7b`         | ~6 GB        | Autocomplete, small scripts, fast replies    |
| `qwen2.5-coder:14b`        | ~10–12 GB    | Default for serious coding & refactors       |
| `qwen2.5-coder:32b`        | ~24–28 GB    | Long reasoning, large repos, best quality    |
| `deepseek-coder-v2:lite`   | ~10–14 GB    | MoE alternative, strong on code reasoning    |

CPU-only? Start with `:14b`. `:32b` works on 64 GB RAM but is slow without a
GPU.

### Profiles

The `pull-models` scripts and `make` targets accept named profiles. Every
profile **skips models that are already installed**, so it is safe to re-run
after adding a new profile without re-downloading the others.

| Profile      | Models pulled                                       | Notes                       |
|--------------|-----------------------------------------------------|-----------------------------|
| `ultralight` | `qwen2.5-coder:1.5b`                                | Smallest                    |
| `fast`       | `qwen2.5-coder:3b`                                  | Snappy autocomplete         |
| `light`      | `qwen2.5-coder:7b`                                  | Low-RAM serious coder       |
| `default`    | `qwen2.5-coder:14b`                                 | Recommended baseline        |
| `heavy`      | `qwen2.5-coder:32b`                                 | Needs lots of RAM           |
| `deepseek`   | `deepseek-coder-v2:lite`                            | MoE coder                   |
| `golang`     | `qwen2.5-coder:3b` + `deepseek-coder-v2:lite`       | Tuned for Go work           |
| `python`     | `qwen2.5-coder:7b` + `deepseek-coder-v2:lite`       | Tuned for Python work       |
| `all`        | every model above                                   | Big disk hit on first run   |

Pull with the helper scripts:

```bash
./scripts/pull-models.sh ultralight
./scripts/pull-models.sh fast
./scripts/pull-models.sh golang
./scripts/pull-models.sh python
./scripts/pull-models.sh all
```

Windows PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\pull-models.ps1 ultralight
powershell -ExecutionPolicy Bypass -File .\scripts\pull-models.ps1 fast
powershell -ExecutionPolicy Bypass -File .\scripts\pull-models.ps1 golang
powershell -ExecutionPolicy Bypass -File .\scripts\pull-models.ps1 python
```

Or via Make:

```bash
make pull-ultralight
make pull-fast
make pull-light
make pull-default
make pull-heavy
make pull-deepseek
make pull-golang
make pull-python
make pull-all
```

### Dry-run

Want to see what a profile would download without touching the network or
Docker? Pass `--dry-run` (bash) / `-DryRun` (PowerShell):

```bash
./scripts/pull-models.sh golang --dry-run
make pull-dry PROFILE=python
```

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\pull-models.ps1 golang -DryRun
```

The script prints `dry-run would pull <model>` for each model in the profile
and exits without contacting the container.

### Custom coder model (Modelfile)

`ollama/Modelfile.coder` defines an `opencodeforge-coder` model: it wraps
`qwen2.5-coder:14b` with a system prompt and code-tuned sampling parameters
(low temperature, 16k context). Build it once after `docker compose up`:

```bash
make build-coder
# or directly:
./scripts/build-coder.sh
```

```powershell
.\scripts\build-coder.ps1
```

The helper auto-pulls the `FROM` base model if it isn't installed yet, then
runs `ollama create opencodeforge-coder -f /modelfiles/Modelfile.coder` inside
the container. After it finishes, pick `opencodeforge-coder` in the Open WebUI
model selector. Edit `ollama/Modelfile.coder` and re-run to update.

Override the name or path:

```bash
MODEL_NAME=my-coder ./scripts/build-coder.sh
```

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

### Edit code in a real editor (Claude-Code-style loop)

OpenCodeForge ships an in-browser VS Code (`code-server`) that mounts the
**same** `./workspace` directory as the Tools API, so the LLM and you edit the
exact same files. It runs on every host that supports Docker (Windows
Docker Desktop / WSL2, Ubuntu, macOS) without installing anything else.

1. `docker compose up -d --build`
2. Open <http://localhost:8443>
3. Login with `CODE_SERVER_PASSWORD` from your `.env` (default
   `opencodeforge`)
4. The chat in Open WebUI (<http://localhost:3000>) calls the Tools API,
   which writes/patches files in the same workspace. The editor picks the
   changes up live.

#### Pointing at any host directory (Windows or Ubuntu)

By default `./workspace` inside the repo is mounted. Set
`WORKSPACE_HOST_DIR` in `.env` to point at any folder on the host:

```env
# Ubuntu / WSL2 / macOS
WORKSPACE_HOST_DIR=/home/me/projects

# Windows Docker Desktop (use forward slashes)
WORKSPACE_HOST_DIR=C:/Users/me/projects

# Windows, but running compose from inside WSL2
WORKSPACE_HOST_DIR=/mnt/c/Users/me/projects
```

Then `docker compose up -d` — `tools-api`, `code-server`, and `indexer` all
see the same tree.

#### Attach your local VS Code via Dev Containers

If you prefer VS Code on the host instead of the browser:

1. Install the **Dev Containers** extension (VS Code on Windows or Ubuntu).
2. `code .` in the repo, then "Dev Containers: Reopen in Container".
3. VS Code attaches to the `code-server` service from
   `.devcontainer/devcontainer.json`, with ports 3000/8088/8443/11434
   forwarded automatically.

#### Let the LLM call the Tools API automatically

`tools-api` serves an OpenAPI 3.1 spec at
<http://localhost:8088/openapi.yaml>. In Open WebUI:

1. Settings → **Tools** → *Add Tool Server*
2. URL: `http://tools-api:8088` (when WebUI runs in compose) or
   `http://localhost:8088`
3. Spec: `http://tools-api:8088/openapi.yaml`
4. Enable the tool in a chat. The model can now call `readFile`,
   `writeFile`, `applyPatch`, `searchWorkspace`, `gitDiff`, and the
   allowlisted `runCommand` — same loop as Claude Code, fully local.

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

## Stop, restart, and resume without re-downloading models

Models live in the **named Docker volume `ollama-data`**, which is independent
from the `ollama` container's lifecycle. As long as that volume is intact, all
models you have already pulled stay on disk and are reused on the next start —
no re-download.

### Safe (keeps models)

```bash
# Stop and remove the containers, KEEP the volumes (default)
docker compose down

# Bring everything back up later — models are still there
docker compose up -d

# Confirm models are still installed
docker exec -t opencodeforge-ollama ollama list

# Re-running pull is safe; it will print "skip ... (already installed)"
./scripts/pull-models.sh default
```

`docker compose stop` (without `down`) is even softer — it only pauses the
containers; `docker compose start` resumes them without rebuilding.

### Destructive (deletes models — avoid)

These remove the `ollama-data` volume and force a fresh download next time:

```bash
docker compose down -v        # the -v wipes named volumes
make reset                    # same thing, with confirmation
./scripts/reset.sh            # same thing, with confirmation
docker volume rm opencodeforge_ollama-data
```

Only use these when you actually want to start over.

### Typical day-to-day loop

```bash
# Morning
docker compose up -d
./scripts/healthcheck.sh

# Work in http://localhost:3000 ...

# End of day
docker compose down           # NOT down -v
```

### Where the model files live

- Volume name: `opencodeforge_ollama-data` (Compose adds the project prefix)
- Mounted inside the container at `/root/.ollama`
- Inspect with: `docker volume inspect opencodeforge_ollama-data`
- Back up by running an `alpine` container that mounts the volume and `tar`s
  `/root/.ollama` to a host path.

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
| Windows: `pull-models.ps1` always re-pulls    | You're on an old copy — pull `main`; the script must call `docker exec` (not `-t`) and parse with `Out-String` |
| Windows: script blocked / `cannot be loaded`  | Run with `powershell -ExecutionPolicy Bypass -File .\scripts\pull-models.ps1 default` |
| Windows: `docker exec` fails with `the input device is not a TTY` | Update to the latest scripts (we removed `-t`); or run from a real PowerShell window, not CI |

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
