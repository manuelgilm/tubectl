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

The output includes the `comment-id` needed for the comment commands below.

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

## Project structure

```
cmd/tubectl/       # CLI entry point and command definitions
internal/youtube/  # YouTube API client, models, auth, and caption parsing
internal/openai/   # OpenAI chat completions client
```

## Data storage

| File | Purpose |
|---|---|
| `~/.config/tubectl/token.json` | OAuth2 token |
| `~/.config/tubectl/transcripts/<videoID>.json` | Cached transcripts |
