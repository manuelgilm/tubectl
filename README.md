# tubectl

A CLI for the YouTube Data API v3, written in Go. Interact with videos, comments, and transcripts from your terminal.

## Requirements

- Go 1.21+
- A Google Cloud project with the **YouTube Data API v3** enabled
- OAuth2 credentials (Desktop app type) from [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials

## Installation

```bash
make install
```

This builds and installs the binary to `$GOPATH/bin`. Make sure that directory is in your `PATH`:

```bash
echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.bashrc && source ~/.bashrc
```

## Configuration

Set your credentials as environment variables. The easiest way is to create a `.env.sh` file in the project root (it is gitignored):

```bash
# .env.sh
export YOUTUBE_CLIENT_ID="your-client-id"
export YOUTUBE_CLIENT_SECRET="your-client-secret"

# Required only for the suggest-reply command
export OPENAI_API_KEY="your-openai-api-key"
```

Load it before running commands:

```bash
source .env.sh
```

## Authentication

```bash
tubectl auth login
```

Opens a Google OAuth2 consent URL. Open it in your browser, grant access, then paste the authorization code back. The token is saved to `~/.config/tubectl/token.json` and reused automatically on subsequent calls. When the token expires it is refreshed silently.

To force a fresh login:

```bash
tubectl auth login --force
```

## Commands

### Get comments for a video

```bash
tubectl video --video-id <videoID> get-comments
```

Filter by publish date:

```bash
tubectl video --video-id <videoID> get-comments --published-after 2024-01-01
```

Output as JSON (useful for scripting and automation):

```bash
tubectl video --video-id <videoID> get-comments --format json
```

The JSON output contains `id`, `author`, `published_at`, and `text` for each comment. You can pipe it into `jq` to extract just the IDs:

```bash
tubectl video --video-id <videoID> get-comments --format json | jq -r '.[].id'
```

The default format is `text`, which includes the `comment-id` needed for the comment commands below.

### Get the transcript of a video

```bash
tubectl video --video-id <videoID> get-transcript
```

Specify a language:

```bash
tubectl video --video-id <videoID> get-transcript --language en
```

Transcripts are cached in `~/.config/tubectl/transcripts/<videoID>.json`. Subsequent calls read from the cache — no API quota is consumed. To bypass the cache:

```bash
tubectl video --video-id <videoID> get-transcript --no-cache
```

> **Note:** The official API only allows downloading captions for videos you own. For other public videos the command automatically falls back to YouTube's public timedtext endpoint.

### Get the content of a comment

```bash
tubectl comment --comment-id <commentID> get-content
```

### Reply to a comment

```bash
tubectl comment --comment-id <commentID> reply "Your reply text"
```

> Requires the `youtube.force-ssl` OAuth scope, which is requested by default during `auth login`.

### AI-suggested reply

Uses the cached transcript as context to generate a reply with OpenAI, then optionally posts it:

```bash
tubectl comment --comment-id <commentID> suggest-reply --video-id <videoID>
```

The command prints the suggestion and asks `[y/N]` before posting. You can skip the prompt:

```bash
# Print the suggestion only — no posting, no prompt
tubectl comment --comment-id <commentID> suggest-reply --video-id <videoID> --print-only

# Post automatically — no prompt (useful in scheduled jobs)
tubectl comment --comment-id <commentID> suggest-reply --video-id <videoID> --auto-approve
```

A cached transcript for the video must already exist. Run `get-transcript` first if it doesn't:

```bash
tubectl video --video-id <videoID> get-transcript
```

Requires `OPENAI_API_KEY` to be set. The model defaults to `gpt-4o-mini`.

## Development

```bash
make build    # compile to ./bin/tubectl
make install  # build and install to $GOPATH/bin
make test     # run all tests
make clean    # remove ./bin/
```

## Automated daily replies (GitHub Actions)

The included workflow (`.github/workflows/reply-comments.yml`) runs every day at 9 AM UTC and automatically replies to new comments on all monitored videos.

### videos.yaml

Add the video IDs you want to monitor to `videos.yaml` at the repo root:

```yaml
videos:
  - ITQioNZ_m_U
  - dQw4w9WgXcQ
```

The workflow iterates over every ID in this file, downloads its transcript once (cached across runs), then calls `suggest-reply --auto-approve` for each comment published in the last 24 hours.

### Required GitHub secrets

| Secret | Description |
|---|---|
| `YOUTUBE_CLIENT_ID` | OAuth2 client ID |
| `YOUTUBE_CLIENT_SECRET` | OAuth2 client secret |
| `OPENAI_API_KEY` | OpenAI API key |
| `TUBECTL_TOKEN` | Base64-encoded `~/.config/tubectl/token.json` |

Generate `TUBECTL_TOKEN` locally after authenticating:

```bash
base64 -w0 ~/.config/tubectl/token.json
```

### Manual trigger

You can also trigger the workflow manually from the **Actions** tab → **Reply to new comments** → **Run workflow**.

## Project structure

```
cmd/tubectl/       # CLI entry point and command definitions
internal/youtube/  # YouTube API client, models, auth, and caption parsing
internal/openai/   # OpenAI chat completions client
videos.yaml        # Video IDs to monitor with the GitHub Actions workflow
```

## Data storage

| File | Purpose |
|---|---|
| `~/.config/tubectl/token.json` | OAuth2 token |
| `~/.config/tubectl/transcripts/<videoID>.json` | Cached transcripts |
