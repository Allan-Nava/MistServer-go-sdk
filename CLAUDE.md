# CLAUDE.md

Go client for the [MistServer](https://mistserver.org) controller API (the JSON API on port 4242,
path `/api`). Small library, no binary: two packages. Go 1.25+ (floor set by golang.org/x/net, pulled in by resty).

## Commands

```bash
make build              # go build ./...  (the root has no Go files, so never `go build .`)
make test               # go vet + go test -race
make lint               # golangci-lint v2
```

CI (`.github/workflows/ci.yml`) runs tidy check, vet, race tests on the go.mod floor and on
stable, golangci-lint and govulncheck. `main` has a ruleset: changes go through a PR with those four
checks green; direct pushes, force-pushes and deletion are refused. Only the repository admin can
bypass, and only when merging a PR — never with a direct push (GitHub doesn't allow a GitHub
Actions bypass on a personal repo). So the contributors
workflow opens a PR instead of pushing; CI doesn't run on PRs created with `GITHUB_TOKEN`, so the
admin merges those by hand. Tags `vX.Y.Z` create a GitHub release with generated notes.
Renovate (not Dependabot) keeps modules and actions current.

## Layout

- `mist/` — the SDK. **Package name is `mist_go`**, directory is `mist`, so callers import it as
  `mist "github.com/Allan-Nava/MistServer-go-sdk/mist"`. Don't rename one without the other — it is
  a breaking change for every importer.
  - `mist.go` — `IMistGoClient` interface (grouped by area), `NewService`, auth, `doAuthorized`,
    `doNoReply` and `postRequest`.
  - `api_streams.go`, `api_stats.go`, `api_push_sessions.go`, `api_config.go`,
    `api_security.go`, `api_variables.go`, `api_raw.go` — one file per area of the API: request
    and response types plus the `*service` methods. The original seven commands still live in
    `request.go` / `response.go`.
  - `configuration.go` — functional options (`WithBaseURL`, `WithUsername`, `WithPassword`).
  - `example_test.go`, `example_commands_test.go` — runnable `Example…` functions (package
    `mist_go_test`), shown on pkg.go.dev and executed by `go test`. The fake server is the
    unexported `apiURL()` helper; add a `cannedReplies` entry for any command with a reply.
  - `commands_test.go` — table of `commandCase`s: exact wire request + decoded reply per method.
  - `request.go` / `response.go` — JSON payloads. Exported request types embed the unexported
    `authorizeRequest`, which the service fills in; callers never set it.
- `lib/util.go` — `GenerateMD5`, used only by the auth handshake.
- `docs/` — GitHub Pages site: static HTML, no build step, no third-party resources (system
  fonts only — keep it that way). Deployed by `.github/workflows/pages.yml` on every push to
  `main` that touches `docs/`; run it by hand with `gh workflow run pages.yml`.

## How a call works

Every public method is one line: `doAuthorized[Resp](s, request)`. It gets the cached login,
sets it on the request (`setAuthorization`, promoted from the embedded `authorizeRequest`),
POSTs the struct as JSON to `BaseUrl`, then checks `authorize.status` in the reply.
MistServer answers **HTTP 200 even when auth fails**, so anything but `OK` resets the cache and
returns `ErrUnauthorized`. Only `CHALL` is retried (once, with a fresh challenge): the challenge is
`md5(date + client host)`, so it rolls over daily. `NOACC` (no accounts on the server) is not
retried — it carries no challenge, and on a local connection the command has already run.

Auth is MistServer's challenge scheme: an empty POST returns `authorize.challenge`, then the password
sent is `md5(md5(password) + challenge)`. The result is cached for one minute behind a mutex
(`lastAuthorized`). MD5 is mandated by the MistServer protocol — don't "upgrade" it.

To add an endpoint, in the `api_*.go` file for its area:
- Request: a struct embedding `authorizeRequest` with the command as its JSON tag. When the
  command takes several shapes (`true` to read vs an object to write, name lists vs field lists),
  expose a plain options struct and build an unexported `…Wire` struct in the method — see
  `StreamTagsRequest` / `streamTagsWire`. Commands with no parameters get an unexported request
  type with a `bool` set to `true`.
- Response: a type embedding `BaseResponse` (that is what makes the status check work). Commands
  MistServer sends no reply member for return only `error`, via `doNoReply`.
- A one-line method on `IMistGoClient` (with a doc comment naming the command) + `*service`.
- A `commandCase` in `commands_test.go` first (red), an `Example<RequestType>` (or
  `Example<ResponseType>` / `ExampleNewService_<name>` when there is no request type), and a row
  in the site's API table. The command names come from
the MistServer API docs: https://docs.mistserver.org/mistserver/integration/api/

## Gotchas

- MistServer reads the POST body as the command **only if `Content-Type` is exactly
  `application/json`** — a charset parameter makes it ignore the body and answer `CHALL`.
  `postRequest` sets the header per request for that reason; don't drop it.
- Replies to `addstream`/`deletestream` put `"incomplete list": 1` inside `streams`, which is why
  `PostStreamResponse.Streams` is `any` and not `map[string]Stream`.
- **The MistServer docs are wrong in places; the controller source is right.** Found so far:
  `browse` takes a path string, not `{"path": …}`; the external writer list comes back as
  `external_writer_list`, not `variable_list`; the log-clearing key is `clearstatlogs`, not
  `clearstatlog`; and the `stop_sessions` array form is read as *protocol* names (the iterator key
  is empty), so `StopSessionsRequest` uses the object form. Check the handler in
  `controller_api.cpp` before trusting a docs page.
- `inject_scte35` is documented but absent from the open-source controller (master and 3.11.2),
  so it has no method; `PostRaw` can send it.
- `streams`, `streamkeys` and `jwks` with a value *replace everything*. `PostStreams` refuses a nil
  map (`ErrInvalidRequest`); for `PostStreamKeys` / `PostJWKS` nil means "read only".

- `NewService` defaults a nil resty client to `resty.New()` and a nil logger to `zap.NewNop()`.
- The logger is a `*zap.SugaredLogger`: key/value calls need the `…w` variants (`Errorw`,
  `Warnw`). Plain `Error("msg", "error", err)` just concatenates.
- **`addstream` / `streams` replace a stream's whole config** (`AddStreams` in
  `controller_streams.cpp` assigns the object as sent). Anything not sent is removed, so every
  other setting goes in `AddStream.Options`. The typed fields are `omitempty` only to avoid sending
  values nobody set; that does *not* preserve anything. `StopSessions` is lifted to a top-level
  `stop_sessions` member, the only place the controller reads it.
- `config_restore` replaces the entire server state (`Storage.assignFrom`); `PostConfigRestore`
  refuses anything that isn't a JSON object.
- `Stream` and `Config` decode leniently (`lenient.go`): the server stores their members
  verbatim, so a wrongly-typed member only zeroes that field instead of failing the whole call.
  Keep new always-sent types lenient too.
- `PostAutoPushStopRequest` is deprecated (duplicate of `PostAutoPushRemoveRequest`); kept only
  because it's exported.
- Raising the Go floor in `go.mod` means updating the first entry of the CI matrix too.

## Conventions

- **TDD, always.** Red first: a test in `mist_test.go` that fails for the right reason, then the
  minimal fix, then refactor. Tests that only pin existing behaviour must be shown to fail on a
  mutation, or they prove nothing.
- **Examples, always.** Every public function, method or behaviour change ships with a runnable
  example in `example_test.go` with an `// Output:` block. Example bodies read like real usage
  (`mist.` qualified, no test helpers beyond `newClient()` / `apiURL()`). Keep README and the
  site's snippets in sync with them.
- **Name examples after something go/doc can attach them to:** a function (`ExampleNewService`,
  `ExampleNewService_health`), a type (`ExamplePostStreamRequest`) or a method declared with a
  receiver. `ExampleIMistGoClient_Health` (interface method) or `ExampleErrUnauthorized` (variable)
  compile, pass vet and run, but pkg.go.dev silently drops them. `TestExamplesAreDocumented`
  (`doc_test.go`) enforces this.
- Server behaviour comes from the MistServer source (`src/controller/controller_api.cpp`,
  `controller_push.cpp`, `controller_streams.cpp` in DDVTech/mistserver), not from guesses.
  `fakeMist` must mirror it: extend the fake before writing a test that depends on new behaviour.

- Keep the public surface on `IMistGoClient`; `service` stays unexported.
- New code gets `go vet` + `golangci-lint` clean. Tests use the `fakeMist` handler in
  `mist_test.go` (challenge round-trip, NOACC, the Content-Type rule) — never a real MistServer.
- The Pages site in `docs/` documents the API by hand — update its endpoint table when the
  interface changes.

## Definition of done

A change is finished only when everything downstream is updated in the same PR, unprompted:

1. Test first, then the fix; an `Example…` for any public API change (see Conventions).
2. Doc comments on what changed; README, `docs/index.html` (endpoint table, quickstart, auth
   section) and this file — grep them for anything the change made stale.
3. `make test && make lint` green, CI green on the PR.
4. After merge and a tag (the user decides when to tag): the release workflow succeeded, the Go
   proxy serves the version, and pkg.go.dev lists it with its examples. If pkg.go.dev 404s on the
   new version, request it: `curl -X POST https://pkg.go.dev/fetch/github.com/Allan-Nava/MistServer-go-sdk@vX.Y.Z/mist`.
