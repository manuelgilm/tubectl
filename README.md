# tubectl

CLI for YouTube management and AI-powered automations.

## Prerequisites

| Variable | Required For |
|---|---|
| `OPENAI_API_KEY` | `tubectl ai` and `tubectl bot` commands |
| `YOUTUBE_CLIENT_ID` | `tubectl auth youtube` (OAuth) |
| `YOUTUBE_CLIENT_SECRET` | `tubectl auth youtube` (OAuth) |

## Installation

```bash
make build       # compile into ./bin/tubectl
make install     # install to $GOPATH/bin/tubectl
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
export OPENAI_API_KEY=...
tubectl bot answer-comment --video-id dQw4w9WgXcQ --comment-id Ug... --only-print

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
  auth/             # OAuth tokens per service
  prompts/          # YAML prompt templates (optional)
```

Created on first use with `tubectl init`.

## Command Reference

### `tubectl init`

Creates the `~/.tubectl/` directory tree (registry, config, transcripts, auth, prompts).

### `tubectl auth`

OAuth authentication for YouTube.

```
tubectl auth youtube           # Authenticate interactively (if no valid token exists)
tubectl auth youtube --force   # Force re-authentication even if a token is valid
```

Opens a browser URL, listens for the OAuth callback on a local port, and saves the token to `~/.tubectl/auth/youtube.json`.

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

## Output Format

All data-fetching commands return JSON. Status messages go to stderr, so you can redirect JSON output to a file:

```bash
tubectl video get --video-id dQw4w9WgXcQ > video.json
```

## Makefile Targets

| Target | Description |
|---|---|
| `make build` | Compile into `./bin/tubectl` |
| `make install` | Install to `$GOPATH/bin/tubectl` |
| `make test` | Run all tests |
| `make clean` | Remove `./bin/` |
