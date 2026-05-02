# Open WebUI integration notes

Open WebUI is the chat interface that talks to Ollama. The container is
defined in `../docker-compose.yml` as `open-webui` and exposed on
`http://localhost:3000` (or `WEBUI_PORT`).

## Configuration

Most settings are configured in-app under **Settings → Connections** and
**Settings → Models**. The defaults wired in compose are:

| Variable             | Default value                  | Why                                  |
|----------------------|--------------------------------|--------------------------------------|
| `OLLAMA_BASE_URL`    | `http://ollama:11434`          | Reach the Ollama service by name.    |
| `WEBUI_AUTH`         | `false`                        | Skip login for single-user setups.   |
| `WEBUI_NAME`         | `OpenCodeForge`                | Branding in the UI.                  |
| `DEFAULT_USER_ROLE`  | `admin`                        | First account is an admin.           |

Set `WEBUI_AUTH=true` in `.env` if you expose the UI beyond localhost.

## Importing prompts

The `../prompts/` directory is bind-mounted into the container at
`/prompts` (read-only). To use a prompt:

1. Open the UI.
2. Go to **Workspace → Prompts → +**.
3. Paste the contents of one of the markdown files from `/prompts`.
4. Save and assign it to a model or use it ad-hoc per chat.

Alternatively, paste the prompt body straight into a new chat as the
system prompt.

## Tools API integration

Open WebUI supports custom tools/functions, but for now the Tools API at
`http://tools-api:8088` is meant to be called by the assistant via
explicit instructions in the system prompt rather than via a function
calling integration. See `../tools-api/prompts/api-usage.md` for the
endpoint reference.
