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
stable, golangci-lint and govulncheck. Tags `vX.Y.Z` create a GitHub release with generated notes.
Renovate (not Dependabot) keeps modules and actions current.

## Layout

- `mist/` — the SDK. **Package name is `mist_go`**, directory is `mist`, so callers import it as
  `mist "github.com/Allan-Nava/MistServer-go-sdk/mist"`. Don't rename one without the other — it is
  a breaking change for every importer.
  - `mist.go` — `IMistGoClient` interface, `NewService`, auth, `doAuthorized` and `postRequest`.
  - `configuration.go` — functional options (`WithBaseURL`, `WithUsername`, `WithPassword`).
  - `request.go` / `response.go` — JSON payloads. Exported request types embed the unexported
    `authorizeRequest`, which the service fills in; callers never set it.
- `lib/util.go` — `GenerateMD5`, used only by the auth handshake.
- `docs/` — GitHub Pages site, served from `main:/docs` (static HTML, `.nojekyll`, no build step).

## How a call works

Every public method is one line: `doAuthorized[Resp](s, request)`. It gets the cached login,
sets it on the request (`setAuthorization`, promoted from the embedded `authorizeRequest`),
POSTs the struct as JSON to `BaseUrl`, then checks `authorize.status` in the reply.
MistServer answers **HTTP 200 even when auth fails**, so anything but `OK` resets the cache,
retries once with a fresh challenge, then returns `ErrUnauthorized`.

Auth is MistServer's challenge scheme: an empty POST returns `authorize.challenge`, then the password
sent is `md5(md5(password) + challenge)`. The result is cached for one minute behind a mutex
(`lastAuthorized`). MD5 is mandated by the MistServer protocol — don't "upgrade" it.

To add an endpoint: request struct embedding `authorizeRequest` with the MistServer command as its
JSON tag (`request.go`), a response type embedding `BaseResponse` if the reply isn't `Response`
(`response.go`) — embedding it is what makes the status check work — and a one-line method on
`IMistGoClient` + `*service` (`mist.go`). Add a case to `mist_test.go`. The command names come from
the MistServer API docs: https://docs.mistserver.org/mistserver/integration/api/

## Gotchas

- `NewService` defaults a nil resty client to `resty.New()` and a nil logger to `zap.NewNop()`.
- The logger is a `*zap.SugaredLogger`: key/value calls need the `…w` variants (`Errorw`,
  `Warnw`). Plain `Error("msg", "error", err)` just concatenates.
- `AddStream` fields are `omitempty`, so zero values are not sent and don't overwrite settings
  on an existing stream. The flip side: you can't explicitly send `DVR: 0`.
- `PostAutoPushStopRequest` is deprecated (duplicate of `PostAutoPushRemoveRequest`); kept only
  because it's exported.
- Raising the Go floor in `go.mod` means updating the first entry of the CI matrix too.

## Conventions

- Keep the public surface on `IMistGoClient`; `service` stays unexported.
- New code gets `go vet` + `golangci-lint` clean. Tests use the `fakeMist` handler in
  `mist_test.go` (it fakes the challenge round-trip) — never a real MistServer.
- The Pages site in `docs/` documents the API by hand — update its endpoint table when the
  interface changes.
