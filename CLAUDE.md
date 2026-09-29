# CLAUDE.md

Go client for the [MistServer](https://mistserver.org) controller API (the JSON API on port 4242,
path `/api`). Small library, no binary: two packages, no tests yet.

## Commands

```bash
go build ./...          # the root has no Go files: `go build .` (and so `make build`) fails
go vet ./...
golangci-lint run ./...
go test ./...           # compiles, but there are no tests
```

## Layout

- `mist/` — the SDK. **Package name is `mist_go`**, directory is `mist`, so callers import it as
  `mist "github.com/Allan-Nava/MistServer-go-sdk/mist"`. Don't rename one without the other — it is
  a breaking change for every importer.
  - `mist.go` — `IMistGoClient` interface, `NewService`, auth, and the generic `postRequest[T, R]`.
  - `configuration.go` — functional options (`WithBaseURL`, `WithUsername`, `WithPassword`).
  - `request.go` / `response.go` — JSON payloads. Exported request types embed the unexported
    `authorizeRequest`, which the service fills in; callers never set it.
- `lib/util.go` — `GenerateMD5`, used only by the auth handshake.
- `docs/` — GitHub Pages site, served from `main:/docs` (static HTML, `.nojekyll`, no build step).

## How a call works

Every public method does the same three things: `getAuthorization()`, copy the auth block into the
request, `postRequest` the whole struct as a JSON body to `BaseUrl`.

Auth is MistServer's challenge scheme: an empty POST returns `authorize.challenge`, then the password
sent is `md5(md5(password) + challenge)`. The result is cached for one minute behind a mutex
(`lastAuthorized`). MD5 is mandated by the MistServer protocol — don't "upgrade" it.

To add an endpoint: request struct embedding `authorizeRequest` with the MistServer command as its
JSON tag (`request.go`), a response type if the reply isn't `Response` (`response.go`), a method on
`IMistGoClient` + `*service` following the existing pattern (`mist.go`). The command names come from
the MistServer API docs: https://docs.mistserver.org/mistserver/integration/api/

## Gotchas

- MistServer answers **HTTP 200 even when auth fails**; the failure is only in
  `authorize.status` (`CHALL`, `NOACC`). `postRequest` does not check it yet.
- `logger` and `restyClient` passed to `NewService` must be non-nil; there is no default.
- The logger is a `*zap.SugaredLogger`: key/value calls need the `…w` variants (`Errorw`,
  `Warnw`). Plain `Error("msg", "error", err)` just concatenates.
- `AddStream` fields have no `omitempty`: zero values (`DVR: 0`, `stop_sessions: false`) are sent.
- `go.mod` says `go 1.19` and CI builds 1.19–1.21, but current resty/zap need newer Go — that is
  why the Renovate dependency PRs fail CI. Raise the Go floor and the CI matrix together.
- `PostAutoPushStopRequest` is unused (duplicate of `PostAutoPushRemoveRequest`).

## Conventions

- Keep the public surface on `IMistGoClient`; `service` stays unexported.
- New code gets `go vet` + `golangci-lint` clean; prefer table tests against `httptest.Server`
  (fake the challenge round-trip) over hitting a real MistServer.
- The Pages site in `docs/` documents the API by hand — update its endpoint table when the
  interface changes.
