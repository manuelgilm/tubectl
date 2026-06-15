# Session

## Done
- `commands.md` — full command spec
- `cmd/init.go` + `cmd/init_test.go` — `tubectl init`
- `internal/auth.go` — `Provider` interface, `Options`, `Status`
- `internal/youtube/` — full `YouTubeProvider` (OAuth, refresh, token storage), API `Client` (get, post, GetVideo, GetComments, ListCaptions, DownloadTranscript, ReplyToComment, DeleteComment, PostComment)
- `internal/youtube/captions_parser.go` — SRT + timedtext XML parsers
- `cmd/auth.go` — `tubectl auth youtube --force`
- `internal/registry/` — CRUD + Load/Save
- `cmd/registry.go` — `tubectl registry add/list/delete/update`
- `cmd/video.go` — `tubectl video get`, `comments`, `get-transcript`
- `cmd/comment.go` — `comment get/reply/delete`
- `cmd/utils.go` — `TubeCtlHome`, transcript cache helpers
- `cmd/root.go` — `loadClient()` helper
- `internal/ai/openai_client.go` — OpenAI client
- `cmd/ai.go` — `tubectl ai complete --model --query`
- `cmd/bot.go` — `tubectl bot answer-comment` (with `--video-id`, `--comment-id`, `--auto-approve`, `--only-print`, `--prompt-file`)

## Code cleanups
- Removed dead `cmd/prompt.go` stub
- Removed unused `--toggle` flag on root command
- Removed commented-out Cobra boilerplate across all `init()` functions
- Fixed `cmd/init_test.go` — `TubeRegistry` missing `registry.` qualifier
- Fixed `cmd/ai.go` — `--query` flag now actually used instead of `text`
- Fixed `internal/youtube/client.go` — API error decode failures now propagate
- Fixed `internal/youtube/auth.go` — unused `opts` parameter marked with `_`

## Next up
- `cmd/bot.go` — remaining: `tubectl bot health/generate/summarize/ask`
