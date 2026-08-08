# Commands

| Command | Description |
|---|---|
| `tubectl init` | Creates `~/.tubectl/` directory tree |
| `tubectl auth` | YouTube OAuth authentication |
| `tubectl registry` | Manage video registrations (local cache) |
| `tubectl video` | Operations with videos |
| `tubectl comment` | Operations with comments |
| `tubectl ai` | Interact with LLMs (OpenAI) |
| `tubectl bot` | YouTube automations powered by AI |
| `tubectl trace` | Inspect traces recorded on the MLflow server |

## auth

`tubectl auth youtube [--force]`

OAuth login. Use `--force` to re-authenticate even if a valid token exists.

## registry

| Subcommand | Flags | Description |
|---|---|---|
| `add` | `--video-id` (req), `--title` | Register a video. Warns if already registered |
| `list` | — | List all registered videos (JSON) |
| `delete` | `--video-id` (req) | Remove a video from the registry |
| `update` | `--video-id` (req), `--title` (req) | Update a registered video's title |

## video

| Subcommand | Flags | Description |
|---|---|---|
| `get` | `--video-id` (req) | Video metadata (title, description, channel, date) |
| `comments` | `--video-id` (req), `--max-results`, `--order` (`time`/`relevance`) | Comment threads (JSON). Default `--max-results`: 20 |
| `get-transcript` | `--video-id` (req), `--language`, `--no-cache`, `--file` | Captions/transcript. Cached locally in SQLite; `--file` keeps them as `transcripts/<video-id>.txt` in the repo (repository as database). Default language: `en` |
| `comment` | `--video-id` (req), `--text` (req) | Post a top-level comment on a video |

## comment

| Subcommand | Flags | Description |
|---|---|---|
| `get` | `--comment-id` (req) | Comment content, author, date (JSON) |
| `reply` | `--comment-id` (req), `--text` (req) | Reply to a comment |
| `delete` | `--comment-id` (req) | Delete a comment |

## ai

`tubectl ai complete --query <text> [--model <model>]`

Send a raw prompt to the LLM. Default model: `gpt-4o-mini`.

## bot

`tubectl bot answer-comment --video-id <id> --comment-id <id> [--auto-approve] [--only-print] [--prompt-file <path>] [--prompt-name <name>]`

Generate an AI reply to a comment using the video transcript as context, then optionally post it.

- By default: shows the reply and prompts for confirmation before posting
- `--auto-approve`: skip confirmation, post immediately
- `--only-print`: generate the reply but do not post
- `--prompt-file`: use a custom YAML prompt template instead of the default
- `--prompt-name`: fetch a prompt from the MLflow registry by name (takes precedence over `--prompt-file`)

## trace

Inspect traces recorded on the MLflow server. Reuses the same credentials as `tubectl prompt` (`tubectl auth mlflow` or env vars).

| Subcommand | Flags | Description |
|---|---|---|
| `get <traceID>` | `--trace-id` | Fetch a single trace (state, inputs/outputs previews, tags, per-span attributes) |
| `list` | `--experiment-id` (default `0`), `--max-results` (default 20, max 500) | List recent traces, newest first |

The trace ID is the MLflow request id (e.g. `tr-4bf92f3577b34da6a3ce929d0e0e4736`). `bot answer-comment` traces are tagged with `source`, `comment_id`, and `video_id`.

> **Note:** `get` uses the MLflow v3 trace API (`/api/3.0/mlflow/traces/get`) for full span data and previews; `list` uses the legacy v2 search API (`/api/2.0/mlflow/traces`) which takes simple `experiment_ids` query params.

## Notes

- All commands return JSON to stdout. Status messages go to stderr, so you can redirect output: `tubectl video get --video-id <id> > video.json`
- `tubectl init` creates `~/.tubectl/`. All other commands create it on demand if missing.
- Directory structure:

```
~/.tubectl/
  registry.json       registered videos metadata
  auth/               stored credentials per service
  config.json         CLI configuration
  prompts/            YAML prompt templates (optional)
```

- Prerequisites:

| Variable | Required for |
|---|---|
| `YOUTUBE_CLIENT_ID` + `YOUTUBE_CLIENT_SECRET` | `tubectl auth youtube` |
| `OPENAI_API_KEY` | `tubectl ai` and `tubectl bot` |
