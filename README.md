# tubectl

CLI for YouTube comment management and AI-powered automations.
`tubectl` fetches video metadata, transcripts, and comments from the
YouTube Data API, caches them locally, and uses OpenAI to generate
intelligent replies. It can run interactively or as a scheduled
GitHub Actions bot to automatically reply to new comments.

## Prerequisites

| Variable | Required For |
|---|---|
| `OPENAI_API_KEY` | `tubectl ai` and `tubectl bot` commands |
| `YOUTUBE_CLIENT_ID` | `tubectl auth youtube` (OAuth) |
| `YOUTUBE_CLIENT_SECRET` | `tubectl auth youtube` (OAuth) |
| `MLFLOW_TRACKING_USERNAME` | `tubectl auth mlflow`, `tubectl prompt`, and MLflow tracing |
| `MLFLOW_TRACKING_PASSWORD` | `tubectl auth mlflow`, `tubectl prompt`, and MLflow tracing |
| `MLFLOW_SERVER_URL` | `tubectl auth mlflow`, `tubectl prompt`, and MLflow tracing (default `https://sandbox-mlflow.gilmanuel.com`) |
| `MLFLOW_TRACING_ENABLED` | MLflow tracing — set to `true` to enable (`ai` / `bot` commands) |
| `MLFLOW_EXPERIMENT_ID` | MLflow tracing — routes traces to a specific experiment (default `"0"`)

## Installation

```bash
# Option 1: Install directly from GitHub (requires Go)
go install github.com/manuelgilm/tubectl@latest

# Option 2: Build from source
make build       # run tests, then compile into ./bin/tubectl
make install     # run tests, then install to $GOPATH/bin/tubectl
```

## Quick Start

```bash
# 1. Initialize the local state directory
tubectl init

# 2. Authenticate with YouTube
export YOUTUBE_CLIENT_ID=...
export YOUTUBE_CLIENT_SECRET=...
tubectl auth youtube

# 3. Register a video
tubectl registry add --video-id dQw4w9WgXcQ --title "Never Gonna Give You Up"

# 4. Get video metadata
tubectl video get --video-id dQw4w9WgXcQ

# 5. Get the transcript
tubectl video get-transcript --video-id dQw4w9WgXcQ

# 6. Fetch comments
tubectl video comments --video-id dQw4w9WgXcQ --max-results 10

# 7. AI-powered reply to a comment
# Use the "id" field from the `video comments` output as --comment-id
export OPENAI_API_KEY=...
tubectl bot answer-comment --video-id dQw4w9WgXcQ --comment-id Ugz... --only-print

# 8. Post the reply (with confirmation prompt)
tubectl bot answer-comment --video-id dQw4w9WgXcQ --comment-id Ug...

# 9. Skip confirmation with --auto-approve
tubectl bot answer-comment --video-id dQw4w9WgXcQ --comment-id Ug... --auto-approve
```

## State Directory

`tubectl` stores all local state under `~/.tubectl/`:

```
~/.tubectl/
  tubectl.db        # SQLite database (videos, transcripts)
  config.json       # CLI configuration
  auth/             # OAuth tokens and credentials per service
  prompts/          # YAML prompt templates (optional)
```

Created on first use with `tubectl init`.

## Command Reference

### `tubectl init`

Creates the `~/.tubectl/` directory tree (config, tubectl.db, auth, prompts).

### `tubectl auth`

Authentication providers for YouTube and MLflow.

```
tubectl auth youtube           # Authenticate interactively (if no valid token exists)
tubectl auth youtube --force   # Force re-authentication even if a token is valid
tubectl auth mlflow --username <user> --password <pass> [--server-url <url>]
tubectl auth mlflow --username <user> --password <pass> [--server-url <url>] --force
```

**YouTube**: Opens a browser URL, listens for the OAuth callback on a local port, and saves the token to `~/.tubectl/auth/youtube.json`. Requires `YOUTUBE_CLIENT_ID` and `YOUTUBE_CLIENT_SECRET` environment variables.

The OAuth redirect URI must be configured in the Google Cloud Console as `http://127.0.0.1:{port}/callback` (the port is chosen at runtime).

**MLflow**: Saves credentials to `~/.tubectl/auth/mlflow.json`. Can also use `MLFLOW_TRACKING_USERNAME` and `MLFLOW_TRACKING_PASSWORD` environment variables instead of flags.

### `tubectl registry`

Local SQLite-backed store for videos you want to monitor.

| Subcommand | Flags | Description |
|---|---|---|
| `add` | `--video-id` (required), `--title` | Register a video |
| `list` | — | List all registered videos (JSON) |
| `delete` | `--video-id` (required) | Remove a video from the registry |
| `update` | `--video-id` (required), `--title` | Update a video's title |

Registering the same video twice returns an error. Deleting a non-existent video returns an error.

### `tubectl video`

Fetch data from the YouTube Data API (requires authentication).

| Subcommand | Flags | Description |
|---|---|---|
| `get` | `--video-id` | Video metadata (title, description, channel, published date) |
| `comments` | `--video-id`, `--max-results` (default 20), `--order` (time/relevance) | Comment threads (JSON) |
| `get-transcript` | `--video-id`, `--language` (default en), `--no-cache`, `--file` | Captions/transcript (cached locally) |
| `comment` | `--video-id`, `--text`, `--auto-approve` | Post a top-level comment on a video |

Transcripts are cached in the local SQLite database (`tubectl.db`). Pass `--no-cache` to skip the cache and always fetch from the API. Pass `--file` to use/store `transcripts/<video-id>.txt` in the repository instead (used by the CI workflow, where the repository acts as the database).

### `tubectl comment`

Operations on individual comments.

| Subcommand | Flags | Description |
|---|---|---|
| `get` | `--comment-id` | Comment content, author, date (JSON) |
| `reply` | `--comment-id`, `--text`, `--auto-approve` | Reply to a comment |
| `delete` | `--comment-id`, `--force` | Delete a comment |

### `tubectl ai`

Low-level interaction with OpenAI.

| Subcommand | Flags | Description |
|---|---|---|
| `complete` | `--query` (required), `--model` | Send a prompt to the LLM and print the response |

Defaults to `gpt-4o-mini`. The model can be overridden with `--model`.

### `tubectl bot`

YouTube automations powered by AI.

| Subcommand | Flags | Description |
|---|---|---|
| `answer-comment` | `--video-id`, `--comment-id`, `--auto-approve`, `--only-print`, `--prompt-file`, `--prompt-name`, `--model`, `--transcript-language` | Generate an AI reply to a comment and optionally post it |

#### `bot answer-comment` flow

1. Fetches the comment by `--comment-id`
2. Loads the transcript for `--video-id` (SQLite cache or API). Use `--transcript-language` to pick a language (default `en`).
3. Builds a prompt with the comment text and transcript as context
4. Calls OpenAI to generate a reply (model defaults to `gpt-4o-mini`; override with `--model`)
5. By default: shows the reply, asks for confirmation, then posts via the YouTube API
6. `--only-print`: just print the reply, don't post
7. `--auto-approve`: skip the confirmation prompt and post immediately
8. `--prompt-file`: use a custom YAML prompt file instead of the default prompt
9. `--prompt-name`: fetch a prompt from the MLflow registry by name (takes precedence over `--prompt-file`)

#### Custom Prompt Files

YAML prompt files should define a `template` string and a `vars` list:

```yaml
template: |
  You are a helpful assistant. Reply to this comment:
  {comment}
  Video context: {transcript}
vars:
  - comment
  - transcript
```

Use with `--prompt-file path/to/prompt.yaml`.

### `tubectl prompt`

Query prompts from the MLflow prompt registry.

| Subcommand | Flags | Description |
|---|---|---|
| `list` | — | List all prompts from the MLflow registry (JSON) |
| `get` | `--name` (required) | Get a specific prompt by name (JSON) |

Requires authentication via `tubectl auth mlflow` or `MLFLOW_TRACKING_USERNAME`/`MLFLOW_TRACKING_PASSWORD` environment variables.

### `tubectl trace`

Inspect traces recorded on the MLflow server. Reuses the same credentials as
`tubectl prompt` (`tubectl auth mlflow` or environment variables).

| Subcommand | Flags | Description |
|---|---|---|
| `get <traceID>` | `--trace-id` | Fetch a single trace (state, inputs/outputs previews, tags, per-span attributes and duration) |
| `list` | `--experiment-id` (default `0`), `--max-results` (default 20, max 500) | List recent traces, newest first |

The trace ID is the "request id" shown in the MLflow UI (e.g.
`tr-4bf92f3577b34da6a3ce929d0e0e4736`):

```bash
tubectl trace list
tubectl trace get tr-4bf92f3577b34da6a3ce929d0e0e4736
```

> **Note:** `get` calls the MLflow v3 trace API
> (`/api/3.0/mlflow/traces/get`) to fetch full span data and previews, while
> `list` uses the legacy v2 search API (`/api/2.0/mlflow/traces`) because it
> accepts simple `experiment_ids` query params; the v3 search endpoint requires
> a serialized `locations` payload and offers nothing extra for listing.

### MLflow Tracing

`tubectl` can trace OpenAI API calls to an MLflow server using the
OpenTelemetry protocol (protobuf). Each `ai complete` or `bot answer-comment`
invocation creates a trace with:

- Model name, prompt messages, and response
- Token usage (prompt, completion, total)
- Latency and finish reason
- Error information on failure
- Trace tags (`bot answer-comment` attaches `source`, `comment_id`, and `video_id`)

Enable it with `MLFLOW_TRACING_ENABLED=true` and configure MLflow credentials
(shared with `tubectl prompt` and `tubectl auth mlflow`):

```bash
export MLFLOW_TRACING_ENABLED=true
export MLFLOW_SERVER_URL=http://localhost:5000   # optional, defaults to https://sandbox-mlflow.gilmanuel.com

# Using environment variables:
export MLFLOW_TRACKING_USERNAME=admin
export MLFLOW_TRACKING_PASSWORD=secret
tubectl ai complete --query "Hello"

# Or using saved credentials:
tubectl auth mlflow --username admin --password secret
tubectl ai complete --query "Hello"
```

Traces appear in the MLflow UI under the experiment specified by `MLFLOW_EXPERIMENT_ID` (default `"0"`). No tracing occurs when
`MLFLOW_TRACING_ENABLED` is unset or not `"true"`. Trace failures (network,
server errors) are logged to stderr and never block the command.

Inspect recorded traces from the CLI with `tubectl trace list` and
`tubectl trace get <traceID>` (see the `tubectl trace` section above).

## Output Format

All data-fetching commands return JSON to stdout. Status and warning messages go to stderr, so you can redirect JSON output to a file:

```bash
tubectl video get --video-id dQw4w9WgXcQ > video.json
```

### Example outputs

`tubectl video get --video-id dQw4w9WgXcQ`:

```json
{
  "id": "dQw4w9WgXcQ",
  "snippet": {
    "title": "Rick Astley - Never Gonna Give You Up",
    "description": "The official video for \"Never Gonna Give You Up\".",
    "publishedAt": "2009-10-25T06:57:33Z",
    "channelId": "UCuAXFkgsw1L7xaCfnd5JJOw"
  }
}
```

`tubectl video comments --video-id dQw4w9WgXcQ --max-results 2`:

```json
[
  {
    "id": "Ugz_abc123",
    "snippet": {
      "videoId": "dQw4w9WgXcQ",
      "topLevelComment": {
        "snippet": {
          "authorDisplayName": "Viewer1",
          "textDisplay": "Great video!",
          "publishedAt": "2026-07-13T10:00:00Z"
        }
      }
    }
  }
]
```

`tubectl video get-transcript --video-id dQw4w9WgXcQ` (transcript lines printed to stdout with timestamps; cache status goes to stderr):

```
[00:00] We're no strangers to love
[00:03] You know the rules and so do I
[00:06] A full commitment's what I'm thinking of
```

`tubectl registry list`:

```json
[
  {
    "ID": "dQw4w9WgXcQ",
    "Title": "Rick Astley - Never Gonna Give You Up",
    "Description": "",
    "ChannelID": "UCuAXFkgsw1L7xaCfnd5JJOw",
    "PublishedAt": "2009-10-25T06:57:33Z",
    "RegisteredAt": "2026-07-13T12:00:00Z",
    "UpdatedAt": "2026-07-13T12:00:00Z"
  }
]
```

## GitHub Actions Auto-Reply

The repository includes a GitHub Actions workflow (`.github/workflows/reply-to-comments.yml`) that runs on a schedule (every 2 hours) and automatically replies to new YouTube comments using the AI bot.

The workflow:

1. **Builds** `tubectl` and restores a base64-encoded OAuth token from a secret.
2. **Fetches comments** for each video listed in `registered_videos.yaml`.
3. **Filters** for comments posted since the last run.
4. **Generates replies** via `tubectl bot answer-comment --auto-approve` for each new comment.
5. **Tracks** which comments have been replied to across runs using `pending.json` artifacts.

Required secrets:

| Secret | Description |
|--------|-------------|
| `YOUTUBE_CLIENT_ID` | YouTube OAuth client ID |
| `YOUTUBE_CLIENT_SECRET` | YouTube OAuth client secret |
| `OPENAI_API_KEY` | OpenAI API key for the AI bot |
| `TUBECTL_TOKEN` | Base64-encoded OAuth token file saved by `tubectl auth youtube` |

To set up: run `tubectl auth youtube` locally, base64-encode `~/.tubectl/auth/youtube.json`, and add it as the `TUBECTL_TOKEN` secret.

## Makefile Targets

| Target | Description |
|---|---|
| `make build` | Run tests, then compile into `./bin/tubectl` |
| `make install` | Run tests, then install to `$GOPATH/bin/tubectl` |
| `make test` | Run all tests (`go test ./...`) |
| `make clean` | Remove `./bin/` |

## Troubleshooting

### `init` not run

If you see errors about missing config or database files, run `tubectl init` first to create the `~/.tubectl/` directory tree.

### OAuth token expired

`tubectl` automatically refreshes expired tokens when making API calls, but the refresh requires `YOUTUBE_CLIENT_ID` and `YOUTUBE_CLIENT_SECRET` to be set in the environment. If refresh fails, re-authenticate with `tubectl auth youtube --force`.

### Transcript not available

Not all YouTube videos have captions enabled. `tubectl` attempts the OAuth-authenticated caption download first, then falls back to the public timedtext endpoint. If both fail, the transcript is reported as unavailable and the bot command continues without transcript context.

### Caption download returns empty body

The caption download endpoint requires the video to be owned by or accessible to the authenticated account. For videos you don't own, `tubectl` falls back to the public timedtext endpoint, which works for most public videos.

### Traces not appearing in MLflow

Check that `MLFLOW_TRACING_ENABLED=true` is set. If it is, verify the MLflow server URL with `MLFLOW_SERVER_URL` — the default of `https://sandbox-mlflow.gilmanuel.com` is a remote server. Tracing uses the same credentials as `tubectl prompt` (env vars or `tubectl auth mlflow`). If traces appear in the wrong experiment, set `MLFLOW_EXPERIMENT_ID` to the correct experiment ID.

## License

Licensed under the [MIT License](LICENSE).
