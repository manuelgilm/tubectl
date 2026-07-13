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
| `MLFLOW_USERNAME` | `tubectl auth mlflow` and `tubectl prompt` commands |
| `MLFLOW_PASSWORD` | `tubectl auth mlflow` and `tubectl prompt` commands |

## Installation

```bash
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
  registry.json     # Registered video metadata
  config.json       # CLI configuration
  transcripts/      # Cached transcripts by video ID
  auth/             # OAuth tokens and credentials per service
  prompts/          # YAML prompt templates (optional)
```

Created on first use with `tubectl init`.

## Command Reference

### `tubectl init`

Creates the `~/.tubectl/` directory tree (registry, config, transcripts, auth, prompts).

### `tubectl auth`

Authentication providers for YouTube and MLflow.

```
tubectl auth youtube           # Authenticate interactively (if no valid token exists)
tubectl auth youtube --force   # Force re-authentication even if a token is valid
tubectl auth mlflow --username <user> --password <pass>
tubectl auth mlflow --username <user> --password <pass> --force
```

**YouTube**: Opens a browser URL, listens for the OAuth callback on a local port, and saves the token to `~/.tubectl/auth/youtube.json`. Requires `YOUTUBE_CLIENT_ID` and `YOUTUBE_CLIENT_SECRET` environment variables.

The OAuth redirect URI must be configured in the Google Cloud Console as `http://127.0.0.1:{port}/callback` (the port is chosen at runtime).

**MLflow**: Saves credentials to `~/.tubectl/auth/mlflow.json`. Can also use `MLFLOW_USERNAME` and `MLFLOW_PASSWORD` environment variables instead of flags.

### `tubectl registry`

Local metadata cache for videos you want to monitor.

| Subcommand | Flags | Description |
|---|---|---|
| `add` | `--video-id` (required), `--title` | Register a video |
| `list` | — | List all registered videos (JSON) |
| `delete` | `--video-id` (required) | Remove a video from the registry |
| `update` | `--video-id` (required), `--title` | Update a video's title |

Registering the same video twice returns a warning. Deleting a non-existent video returns an error.

### `tubectl video`

Fetch data from the YouTube Data API (requires authentication).

| Subcommand | Flags | Description |
|---|---|---|
| `get` | `--video-id` | Video metadata (title, description, channel, published date) |
| `comments` | `--video-id`, `--max-results` (default 20), `--order` (time/relevance) | Comment threads (JSON) |
| `get-transcript` | `--video-id`, `--language` (default en), `--no-cache` | Captions/transcript (cached locally) |
| `comment` | `--video-id`, `--text` | Post a top-level comment on a video |

Transcripts are cached to `~/.tubectl/transcripts/{video-id}.json`. Pass `--no-cache` to skip the cache and always fetch from the API.

### `tubectl comment`

Operations on individual comments.

| Subcommand | Flags | Description |
|---|---|---|
| `get` | `--comment-id` | Comment content, author, date (JSON) |
| `reply` | `--comment-id`, `--text` | Reply to a comment |
| `delete` | `--comment-id` | Delete a comment |

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
| `answer-comment` | `--video-id`, `--comment-id`, `--auto-approve`, `--only-print`, `--prompt-file` | Generate an AI reply to a comment and optionally post it |

#### `bot answer-comment` flow

1. Fetches the comment by `--comment-id`
2. Loads the transcript for `--video-id` (cache or API)
3. Builds a prompt with the comment text and transcript as context
4. Calls OpenAI to generate a reply
5. By default: shows the reply, asks for confirmation, then posts via the YouTube API
6. `--only-print`: just print the reply, don't post
7. `--auto-approve`: skip the confirmation prompt and post immediately
8. `--prompt-file`: use a custom YAML prompt file instead of the default prompt

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

Requires authentication via `tubectl auth mlflow` or `MLFLOW_USERNAME`/`MLFLOW_PASSWORD` environment variables.

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
{
  "videos": [
    {
      "title": "Rick Astley - Never Gonna Give You Up",
      "video_id": "dQw4w9WgXcQ",
      "published_at": "2009-10-25T06:57:33Z",
      "registered_at": "2026-07-13T12:00:00Z"
    }
  ]
}
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

If you see errors about missing registry or config files, run `tubectl init` first to create the `~/.tubectl/` directory tree.

### OAuth token expired

`tubectl` automatically refreshes expired tokens when making API calls, but the refresh requires `YOUTUBE_CLIENT_ID` and `YOUTUBE_CLIENT_SECRET` to be set in the environment. If refresh fails, re-authenticate with `tubectl auth youtube --force`.

### Transcript not available

Not all YouTube videos have captions enabled. `tubectl` attempts the OAuth-authenticated caption download first, then falls back to the public timedtext endpoint. If both fail, the transcript is reported as unavailable and the bot command continues without transcript context.

### Caption download returns empty body

The caption download endpoint requires the video to be owned by or accessible to the authenticated account. For videos you don't own, `tubectl` falls back to the public timedtext endpoint, which works for most public videos.

## License

Licensed under the [MIT License](LICENSE).
