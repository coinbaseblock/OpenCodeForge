# Tools API quick reference

The tools-api is a small HTTP service that the assistant can call to interact
with the workspace. It is the only path through which the LLM should touch
files or run commands.

## Endpoints

| Method | Path           | Purpose                                          |
|--------|----------------|--------------------------------------------------|
| GET    | `/health`      | Liveness probe.                                  |
| GET    | `/files`       | List a directory (`?path=relative`).             |
| GET    | `/read`        | Read a single file (`?path=relative`).           |
| POST   | `/write`       | Write a file. Body: `{path, content, mode}`.     |
| POST   | `/patch`       | Apply a unified diff. Body: `{diff, strip, cwd}`.|
| GET    | `/search`      | Substring search (`?q=text&path=relative`).      |
| GET    | `/git/status`  | `git status --porcelain` in the workspace.       |
| GET    | `/git/diff`    | `git diff` (`?staged=true&file=...`).            |
| POST   | `/run`         | Run an allowlisted shell command.                |
| GET    | `/run/policy`  | Inspect the current command policy.              |

## Conventions

- Every `path` is relative to `/workspace`.
- `..` and absolute paths are rejected with `403`.
- `/read` returns the file as a JSON string. Files larger than 2 MiB are rejected.
- `/write` creates parent directories. Use `mode: "create"` to refuse overwrite.
- `/patch` shells out to GNU `patch -p<strip> --batch --forward`. The diff
  must be a real unified diff with `a/` and `b/` prefixes.
- `/run` runs `sh -c <cmd>` inside the workspace. Default timeout is 5 minutes,
  capped at 15 minutes.

## Example

```bash
curl -s http://localhost:8088/files?path=. | jq
curl -s -X POST http://localhost:8088/write \
  -H 'Content-Type: application/json' \
  -d '{"path":"hello.txt","content":"hi\n"}'
curl -s -X POST http://localhost:8088/run \
  -H 'Content-Type: application/json' \
  -d '{"cmd":"go test ./...","cwd":"project"}'
```
