# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# radio-jockey

A Discord-bot-driven internet radio station. Users queue YouTube tracks via Discord slash commands; the server streams them to Icecast; listeners tune in via a web player.

## Architecture

```
Discord bot (TS/Bun)
    ↕ Connect-RPC (HTTP/1.1 + JSON)
Go server (connect-go)
    ↕ TCP / HTTP PUT (ICY protocol)
Icecast server
    ↕ HTTP stream (OGG/Opus)
SvelteKit web app  ←  browser audio player
```

Caddy sits in front and routes traffic. Sessions are in-memory only — server restart loses all sessions. Session *archives* (which tracks actually played) are persisted in SQLite — see below.

## Services

| Directory | Tech | Role |
|---|---|---|
| `server/` | Go, connect-go | RPC server + audio pipeline |
| `discord-jockey/` | TypeScript, Bun, discord.js | Discord bot |
| `web/` | SvelteKit, Bun | Frontend — all routes implemented |
| `icecast/` | Icecast2 | Audio relay server |
| `caddy/` | Caddy | Reverse proxy / TLS |

## Build tooling

Commands are run via `just` (see `justfile` at repo root): `just dev`, `just codegen`, `just logs <service>`, `just clean`, `just prune`, `just genkeys`, `just signnonce <nonce>`, `just server-test`. Bare `just` lists all recipes (`default: @just --list`).

- **`just dev`** — `docker compose` (base + `docker-compose.dev.yml`) up with `--watch`, tailing all logs. This is the normal dev loop; requires Docker running.
- **`just prod`** — `docker compose up --build -d` against the base compose file only (no dev overrides).
- **`just codegen`** — regenerates protobuf/connect stubs for all three consumers (server, discord-jockey, web) from `proto/radio-jockey.proto` via the `codegen` compose profile. Run this after editing the proto file.
- **Server tests** — `just server-test` runs the full Go suite in Docker (see Testing section below). To run a subset, override the container command, e.g. `docker compose --profile test run --rm --build server-test go test ./test/session/... -run TestSkip -v`.
- **Web lint/typecheck** — `cd web && bun run lint` (prettier --check + eslint), `bun run format` (prettier --write), `bun run check` (svelte-kit sync + svelte-check). No web test suite exists.
- **Web dev server (standalone)** — `cd web && bun run dev` for iterating on `web/` alone without the full Docker stack (RPC calls will fail without a running `server`).

## Proto API (`proto/radio-jockey.proto`)

All RPCs are unary. `DeleteSessionAuth` and `DeleteSessionArchive` require a Bearer token.

| RPC | Auth | Purpose |
|---|---|---|
| `Ping` | — | health check |
| `RequestNonce` | — | step 1 of auth handshake |
| `RespondNonce` | — | step 2; returns PASETO token |
| `CreateSession` | — | creates session + starts Icecast mountpoint; optional `archive` flag persists a session archive header row |
| `GetSession` | — | returns stream URL for existing session |
| `DeleteSessionAuth` | ✅ Bearer | tears down session |
| `AddTrack` | — | downloads YouTube URL, enqueues |
| `RemoveTrack` | — | removes by queue index (index 0 = currently playing, rejected) |
| `SkipTrack` | — | skips current track |
| `ListQueue` | — | returns all queued tracks (index 0 = now playing) |
| `ListSessions` | — | enumerates active sessions (session_id + stream_url) |
| `ListSessionArchives` | — | enumerates archived session runs (id + session_id + created_at) |
| `GetSessionArchive` | — | returns an archive's header + the tracks that actually played, in play order |
| `DeleteSessionArchive` | ✅ Bearer | deletes an archive header and its child track rows |

`Track` messages carry `album_art_url` (see Server section) alongside id/source/source_id/title/artist/duration.

## Server (`server/src/`)

- **`main.go`** — wires everything together, starts `icecast.StreamSessions` in a goroutine
- **`connect/server.go`** — HTTP server setup with connect-go interceptors; `Server` struct holds `db *db.DB` (used directly by the archive RPCs)
- **`connect/rpc.go`** — all RPC handler implementations
- **`connect/interceptor.go`** — four interceptors: `rateLimitInterceptor`, `loggingInterceptor`, `stripInterceptor` (trims whitespace from all string fields), `authInterceptor` (gates `DeleteSessionAuth` and `DeleteSessionArchive` via `AuthenticatedProcedures`)
- **`connect/ratelimit.go`** — per-IP token-bucket rate limiting (`golang.org/x/time/rate`, configured via `RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`) gating `CreateSession`/`AddTrack`/`RemoveTrack`/`SkipTrack` (`RateLimitedProcedures`). Keys by the last entry in `X-Forwarded-For` — Caddy appends rather than replaces that header, so the last entry is the one Caddy itself added and can't be spoofed by the client — falling back to the raw TCP peer address when absent (e.g. tests hitting the server directly). A background goroutine periodically purges idle per-IP entries so the map doesn't grow unbounded.
- **`connect/auth/auth.go`** — PASETO v4 asymmetric auth. Nonce flow: server issues random nonce → caller signs it with shared private key → server verifies + issues token (24h TTL). Nonces expire in 2 min.
- **`session/session_manager.go`** — in-memory map of `SessionID → *SessionQueue`. `CreateSession(ctx, sessionId, archiveID *int64)` threads an optional archive header ID through to the queue and rejects new sessions past `MAX_SESSIONS` concurrent sessions (`TooManySessionsError`). Emits `SessionCreated`/`SessionDeleted` events on a channel. `ListSessions()` returns all active session IDs. `InUseTrackIDs()` returns the set of track IDs currently playing or queued across every session, so the track cache (below) knows what it must not evict.
- **`session/session_queue.go`** — per-session queue (max 16). Carries an optional `archiveID *int64` (exposed via `ArchiveID()`) set at creation. Operations: `Enqueue`, `Dequeue`, `Peek`, `Remove`, `Skip`, `ListQueue`. Skip fires a `SkipTrack` event; `Notify()` channel wakes the stream loop when a track is added to an empty queue.
- **`session/session_event.go`** — event type definitions
- **`icecast/icecast.go`** — listens for session events and manages one ffmpeg pipeline per session. Audio flow: `ffmpeg -i <opus file> → PCM pipe → ffmpeg (libopus encoder) → Icecast TCP connection`. Streams silence when queue is empty; auto-ends session after 10 min of silence. Holds a `db *db.DB` reference; the instant a track actually starts streaming (not just when enqueued), if the queue has an `ArchiveID()`, it writes a `session_archive_tracks` row via `db.AddSessionArchiveTrack`.
- **`music/youtube.go`** — downloads YouTube audio via yt-dlp, stores files under `music/youtube/`, stores metadata in SQLite. `IsYouTubeURL` rejects non-YouTube hosts before any yt-dlp invocation. `DownloadTrackFromURL` first resolves the video id cheaply via `peekSourceID` (`yt-dlp --skip-download --print id`) and checks the in-memory `Cache` (see `music/cache.go`) for a hit — if the cached file still exists on disk it's returned without downloading; otherwise it falls through to the real `ytdlp()` download. Computes `album_art_url` as `https://i.ytimg.com/vi/<video_id>/hqdefault.jpg` (YouTube's thumbnail CDN — no extra download or serving infra needed). On a fresh download this is passed to `CreateTrack`; on the "track already exists" fast path it's refreshed via `db.UpdateTrackAlbumArtUrl` rather than left stale.
- **`music/cache.go`** — bounds how many downloaded `.opus` files live on disk. `Cache` keeps an in-memory LRU window (`CacheWindow = 50` tracks) mirroring `tracks.last_used_at`, hydrated from the DB on startup via `ListTracksByLastUsed` so a restart doesn't forget recency. `Touch` (called from `AddTrack` in `connect/rpc.go`) bumps a track to the front and, once over the window, deletes the oldest file that isn't in `SessionManager.InUseTrackIDs()` (yt-dlp re-downloads on a missing file, so deleting is safe — the DB row is kept). `Get` checks a source id without writing to the DB, only reordering the in-memory window, so a lookup ahead of a real download can't itself be evicted before the caller re-`Touch`es it.
- **`db/`** — SQLite via sqlx + goose migrations.
  - `tracks` (id, source, source_id, title, artist, duration, file_path, created_at, `album_art_url`, `last_used_at`)
  - `session_archives` — one header row per `CreateSession` call with `archive=true` (surrogate-keyed log of runs, not keyed by session_id since names aren't unique over time)
  - `session_archive_tracks` — one row per track that actually started playing, referencing `tracks.id`; SQLite FKs aren't enforced (no `PRAGMA foreign_keys`), so `DeleteSessionArchive` manually deletes child rows before the header row

**Networking gotcha — `STREAM_BASE_URL`:** this env var (server/.env) is used to build the `stream_url` returned to RPC callers, and it must be reachable by whoever's consuming the URL. The browser (host machine, via Caddy) needs something like `https://localhost:9999` (see `caddy/Caddyfile.dev`'s dedicated `localhost:9999` block); a container on the docker-compose network (like the Discord bot) can't resolve `localhost` to that — it needs the actual service hostname (`http://icecast:9999`, direct to the Icecast container, no TLS). One value can't satisfy both, which is why the Discord bot no longer trusts the RPC's echoed `streamUrl` — see below. Album art URLs don't have this problem since they're public `i.ytimg.com` links reachable from anywhere.

**Docker gotcha:** `env_file` values are only re-read on container *recreation* (`docker compose up -d <service>`), not on `docker compose restart <service>`. If you change a `.env` file, use `up -d --build <service>`.

**Capacity/abuse limits (`server/.env`):** `MAX_SESSIONS` (default 50) caps concurrent sessions server-wide, independent of the per-session 16-track queue cap. `RATE_LIMIT_RPS`/`RATE_LIMIT_BURST` (defaults 1/5) configure the per-IP token bucket in `connect/ratelimit.go`. `music.CacheWindow` (50 tracks) is a Go constant, not an env var.

## Testing (`server/test/`)

Mirrors `server/src/`'s package layout, using external test packages (`db_test`, `session_test`, `icecast_test`, `integration_test`) against the exported API only:

- **`test/db/`**, **`test/session/`**, **`test/icecast/`**, **`test/music/`** — focused unit tests, one behavior per test function. `test/music/` covers `IsYouTubeURL` host matching and `Cache` (window eviction, in-use protection, `Get` hit/miss/reorder, hydration from `last_used_at` on a simulated restart).
- **`test/integration_test.go`** — boots the real HTTP server, real RPC handlers, real `SessionManager`, and a real `IcecastClient` against a **fake Icecast TCP stand-in** (`test/util.FakeIcecastServer`) that speaks the actual PUT/handshake protocol. This exercises real ffmpeg encode/decode and real archive-on-play recording without mocking Go code or needing a real Icecast server — only the external Icecast *service* is stood in for. Tests that don't care about audio use a lightweight ready-ack stand-in instead of live streaming (a live consumer would otherwise race against queue assertions — see `TestQueueOperations`'s history for why).
- **`test/util/util.go`** — shared test helpers: `OpenTestDB`, `GenerateSilentOpus` (renders a short silent `.opus` via ffmpeg for tests needing a real decodable file), `TestAuth`/`SignNonce` (throwaway PASETO keypair, signs nonces like the Discord bot's `signnonce` flow), `FakeIcecastServer`.
- **Not covered:** `AddTrack`'s yt-dlp download path — tests seed tracks directly into the DB/queue instead of hitting the real network.
- **Run via Docker, not the host** — `just server-test` (`docker compose --profile test run --rm --build server-test`), backed by a `test` stage in `server/Dockerfile` (refactored into `base` → `builder`/`test`) with ffmpeg installed. Nothing in the suite depends on the host having Go or ffmpeg.

## Discord Bot (`discord-jockey/src/`)

- **`connect/client.ts`** — Connect-RPC client pointed at `SERVER_HOST:SERVER_PORT`
- **`connect/auth/auth.ts`** — auth helpers: signs nonce with `paseto-ts`, caches token per session, auto-refreshes on 401
- **`discord/sessions.ts`** — in-memory map of `guildId → { connection, buffer, authToken }`. One session per guild.
- **`discord/command/play.ts`** — main command. Creates or joins session, connects to voice channel, streams Icecast HTTP into `PassThrough` buffer (20 MB), feeds to discord.js audio player. On player idle/error, reconnects stream. Builds its own stream URL locally from `ICECAST_INTERNAL_URL` (`http://icecast:9999`, docker-network-internal) rather than trusting the server's `streamUrl` response, which is meant for external/browser consumption (see Networking gotcha above). Always passes `archive: true` to `createSession` — bot-created stations are archived by default.
- **`util/helpers.ts`** — `getSessionId(interaction)` builds `<server-name-slug>-<guildId>` as the RPC session ID; used by every command instead of the bare guild ID.
- **Commands:** `play` (+ optional URL), `queue`, `skip`, `remove <index>`, `stop`, `ping`

**Auth note:** The Discord bot holds `PRIVATE_PASETO_KEY`. It's the only caller that can sign nonces automatically and thus the only caller that can call `DeleteSessionAuth`/`DeleteSessionArchive` without manual steps. The web admin page (see below) can still get a token, but requires the operator to manually run the `signnonce` CLI.

## Web (`web/src/`)

- **`routes/+layout.svelte`** — global layout; on mount, lazy-imports `playhtml` and calls `playhtml.init({ room: 'site', cursors: {...} })` with a static (not path-derived) room name so every page shares one presence room sitewide — shared cursors (with chat) are cosmetic, but the same room also backs `VisitorCount` (below), so this isn't purely decorative. Cursors render into a dedicated fixed+`overflow:hidden` container (not `document.body`) so an out-of-viewport cursor is clipped immediately instead of causing a scrollbar flash.
- **`lib/connect/client.ts`** — Connect-RPC client with empty `baseUrl` (uses same origin via Caddy proxy)
- **`lib/proto/`** — generated protobuf + connect TypeScript stubs
- **`lib/format.ts`** — `formatTimestamp(unixSeconds: bigint)`, formats as `YYYY-MM-DD-HH:MM:SSZ` in UTC. Shared by every page that displays a `created_at`.
- **`lib/components/`** — shared Svelte components, extracted to avoid repetition across the list-heavy pages:
  - `TrackListItem.svelte` — album art thumbnail + title/artist, optional `href` (wraps text in a link, used by the archive view) and `children` snippet (trailing content, e.g. a remove button in the live queue)
  - `LoadingButton.svelte` — `{loading, label, onclick?, type?}`, renders `label` or `"loading..."` and disables itself while `loading`
  - `EntryList.svelte` — generic `{items, emptyMessage, key, item}` snippet-based list; renders `emptyMessage` when empty, otherwise a `<ul>` of `item(entry)` snippets. Takes an optional `class` for callers that want the `>`-bullet style (`:global(.arrow-list)`, defined per-page since not every list uses it)
  - `NotFound.svelte` — `{message, backHref, backLabel}`, used by both `[sessionId]` and `[archiveId]` not-found states
  - `VisitorCount.svelte` — joins the same sitewide `playhtml` presence room as the cursor effects (`+layout.svelte`) via `playhtml.presence.setMyPresence('online', true)`, counts live presences via `onPresenceChange`. No server involvement — presence is peer-to-peer through playhtml's own backend, not the RPC server.
  - `YouTubeSearchBar.svelte` / `YouTubeSearchResults.svelte` / `YouTubeSearch.svelte` — YouTube search for queueing tracks on `/stations/[sessionId]`, alongside (not replacing) the manual paste-a-URL form. `YouTubeSearch.svelte` owns all state (query, paginated results, in-flight add IDs) and composes the other two: `YouTubeSearchBar` is just the submit-triggered (not live-as-you-type, to conserve quota) input, `YouTubeSearchResults` renders results via the existing `EntryList`/`TrackListItem` pair (so it looks identical to every other list in the app) with a `< prev`/`next >` pager backed by the YouTube API's `pageToken` cursor. Each result's "add" button calls the *existing* `AddTrack` RPC with a synthesized `https://www.youtube.com/watch?v=<id>` URL — no proto/server changes were needed. `TrackListItem`'s `track` prop was loosened to `Pick<Track, 'title' | 'artist' | 'albumArtUrl'>` so search results (no DB `id`/`duration` yet) can reuse it.
- **`routes/api/youtube-search/+server.ts`** — SvelteKit server route (not a Go RPC) that proxies YouTube Data API v3 `search.list`. Exists so the `YOUTUBE_API_KEY` never reaches the browser: it's a private env var (`web/.env`, `$env/dynamic/private`, no `PUBLIC_` prefix), read at request time rather than baked in at build — required because `web/Dockerfile`'s build stage receives no env vars at all (`env_file` in `docker-compose.yml`/`docker-compose.dev.yml` only injects into the running container). Trims YouTube's response down to `{videoId, title, channelTitle, thumbnailUrl}` per result plus `nextPageToken`/`prevPageToken`; returns `503` if the key is unset so the UI can show "not configured" instead of failing silently. No rate limiting on this route (matches the project's no-auth-for-DJ-actions posture) — get a key via Google Cloud Console (enable "YouTube Data API v3", create an API key).
- **`lib/errors.ts`** — `friendlyError(err)` (generic fallback message) and `addTrackErrorMessage(err)` (maps `AddTrack`'s `ConnectError` codes to user-facing text), shared by the manual URL form and YouTube search's "add" buttons since both call the same RPC.

**Loading convention:** every button that triggers an RPC shows `"loading..."` while in flight (via `LoadingButton`), and every page's initial list/data fetch shows a `"loading..."` message until that first fetch resolves (via a `loaded` boolean gating the real content) — never silently render an empty/default state while a request is still in flight. Per-row actions in a list (stop a session, delete an archive, remove a queue index) track a `Set` of in-flight keys so only the clicked row's button reflects that specific request.

### Routes

| Route | File | Status |
|---|---|---|
| `/` | `routes/+page.svelte` | Links to `/stations`, `/stations/create`, `/archive` |
| `/stations` | `routes/stations/+page.svelte` | Lists active sessions via `ListSessions`, links to each |
| `/stations/create` | `routes/stations/create/+page.svelte` | Form to name + create a session (with an opt-in "archive this session" checkbox, default off), redirects to it |
| `/stations/[sessionId]` | `routes/stations/[sessionId]/+page.svelte` | Custom play/stop + volume player, now-playing (with album art), queue (polled, with per-track thumbnails), add via YouTube search (paginated) or manual URL paste, skip/remove |
| `/admin` | `routes/admin/+page.svelte` | Nonce/passkey login (token cached in `localStorage`), lists + force-stops active sessions, lists + deletes archived sessions |
| `/archive` | `routes/archive/+page.svelte` | Lists all session archives via `ListSessionArchives`, links to each |
| `/archive/[archiveId]` | `routes/archive/[archiveId]/+page.svelte` | Shows an archive's header + played tracks (with album art); YouTube-sourced tracks link straight to the source video |

## Product decisions

- **Web is full DJ** — any web user can create sessions, add/skip/remove tracks, no auth required. This is a toy project; the admin page is the only control lever.
- **`/stations/create`** — web-based session creation. Works without Discord because the Go server owns the Icecast pipeline; the browser just connects to the stream URL directly as a listener.
- **Admin page** — server-owner view: list all active sessions, force-stop any session; list + delete archived sessions. This is the only auth-gated surface. Since only the Discord bot holds `PRIVATE_PASETO_KEY`, the admin UI walks the operator through the existing nonce flow manually: request a nonce in the browser, run `go run ./cmd/signnonce <nonce>` (or `just signnonce <nonce>`) locally where `PRIVATE_PASETO_KEY` is set, paste the signed passkey back in to get a bearer token.
- **Session names** — `session_id` is caller-defined and serves as the display name. Discord bot passes `<servername>-<guildid>` (via `getSessionId()` in `discord-jockey/src/util/helpers.ts`) instead of bare guild ID. Web users pick their own session ID string (slugified client-side before hitting the RPC).
- **Web auth** — none for DJ actions (add/skip/remove), open to any web user. Admin requires auth via the manual nonce flow described above.
- **Session archives** — opt-in per session (Discord bot always archives; web defaults off). Archives are a surrogate-keyed log of runs (not keyed by session_id, which can be reused over time) and record only tracks that actually started streaming, not everything momentarily queued. Track links in the archive view point straight to the source YouTube URL rather than a new internal track page.
- **Album art** — sourced from YouTube's public thumbnail CDN (`i.ytimg.com/vi/<id>/hqdefault.jpg`) rather than downloading/embedding/self-hosting an image, since every track's source is currently YouTube and this needs zero new infrastructure or `STREAM_BASE_URL`-style browser-vs-docker URL wiring.

## Deployment

- **`deploy.sh`** — pushes the current branch to a VPS (`root@radiojockey.live`) and runs `just prod` there over SSH. It clones/pulls `$DEPLOY_PATH` to match the local branch, then copies local env files up (notably `server/.env.production` → remote `server/.env` — the only file in the pair that's renamed, since local dev and prod need different `server/.env` contents). Run manually (`./deploy.sh`); nothing in CI triggers it.
- **Startup ordering** — `server` exposes a Docker healthcheck (`nc -z localhost 12625`); `caddy`, `web`, and `discord-jockey` all `depends_on: server: condition: service_healthy`, so they won't start handling traffic until the RPC server's port is actually listening.
- **Prod vs dev Caddy config** — `caddy/Caddyfile` (prod) routes real domains (`radiojockey.live`, `stream.radiojockey.live`); `caddy/Caddyfile.dev` (used by `just dev` via the dev compose override) routes `localhost:3000`/`localhost:9999` with `tls internal` for local HTTPS. Same route shapes, different hosts/TLS — see the `STREAM_BASE_URL` gotcha above for why the stream host in particular can't be a single shared value.

## Known remaining gaps

- **Playback position** — no elapsed time exposed. Browser joins the live Icecast stream mid-track with no progress indicator. Acceptable for now.
- **Session persistence** — live session state (queue, playback) is in-memory only. Server restart kills active sessions, though session *archives* (past runs) survive since they're in SQLite.
