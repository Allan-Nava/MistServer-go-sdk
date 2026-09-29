package mist_go

import (
	"encoding/json"
	"fmt"
)

// ConfigRequest changes core server settings ("config"). Only the members
// given are changed. null resets debug and triggers to their defaults; for
// the session modes it is stored as-is and read as 0 until a restart. "protocols" replaces all
// outputs — use PostAddProtocol / PostDeleteProtocol / PostUpdateProtocol to
// change them one at a time.
type ConfigRequest struct {
	authorizeRequest
	Config map[string]any `json:"config"`
}

// Protocol is one output configuration: "connector" plus its settings, as
// listed by PostCapabilities.
type Protocol map[string]any

// AddProtocolRequest enables outputs without touching the others
// ("addprotocol"). An output configured identically already is skipped.
type AddProtocolRequest struct {
	authorizeRequest
	AddProtocol []Protocol `json:"addprotocol"`
}

// DeleteProtocolRequest removes outputs that match exactly ("deleteprotocol").
type DeleteProtocolRequest struct {
	authorizeRequest
	DeleteProtocol []Protocol `json:"deleteprotocol"`
}

// UpdateProtocolRequest replaces every output that exactly matches Old with
// New ("updateprotocol").
type UpdateProtocolRequest struct {
	Old Protocol
	New Protocol
}

type updateProtocolWire struct {
	authorizeRequest
	UpdateProtocol [2]Protocol `json:"updateprotocol"`
}

type configBackupRequest struct {
	authorizeRequest
	ConfigBackup bool `json:"config_backup"`
}

// ConfigBackupResponse holds the full configuration, exactly as it would be
// written to the config file. Feed it back through PostConfigRestore.
type ConfigBackupResponse struct {
	BaseResponse
	ConfigBackup json.RawMessage `json:"config_backup"`
}

// ConfigRestoreRequest replaces the full configuration ("config_restore"):
// accounts, config, streams, pushes, keys and variables all come from
// ConfigRestore, which must be a JSON object such as a PostConfigBackup
// reply. Anything else returns ErrInvalidRequest without contacting the
// server, since MistServer would wipe its configuration. It runs before any
// other command in the same request.
type ConfigRestoreRequest struct {
	authorizeRequest
	ConfigRestore json.RawMessage `json:"config_restore"`
}

type saveRequest struct {
	authorizeRequest
	Save bool `json:"save"`
}

// UISettingsRequest stores arbitrary interface settings in the server config
// ("ui_settings"). A nil map only reads them.
type UISettingsRequest struct {
	UISettings map[string]any
}

type uiSettingsWire struct {
	authorizeRequest
	UISettings any `json:"ui_settings"`
}

// UISettingsResponse holds the stored settings.
type UISettingsResponse struct {
	BaseResponse
	UISettings json.RawMessage `json:"ui_settings"`
}

type apiEndpointRequest struct {
	authorizeRequest
	APIEndpoint bool `json:"api_endpoint"`
}

// APIEndpointResponse is the local URL of the TCP API, e.g.
// "http://127.0.0.1:4242/".
type APIEndpointResponse struct {
	BaseResponse
	APIEndpoint string `json:"api_endpoint"`
}

// BrowseRequest lists a directory on the server ("browse"). Browse is a path,
// absolute or relative to the controller's working directory; empty means the
// working directory.
type BrowseRequest struct {
	authorizeRequest
	Browse string `json:"browse"`
}

// BrowseResult is a directory listing. Path holds the resolved absolute path
// as its only element; use Dir.
type BrowseResult struct {
	Path           []string `json:"path"`
	Files          []string `json:"files"`
	Subdirectories []string `json:"subdirectories"`
}

// Dir returns the resolved directory path.
func (b BrowseResult) Dir() string {
	if len(b.Path) == 0 {
		return ""
	}
	return b.Path[0]
}

// BrowseResponse is the reply to PostBrowse.
type BrowseResponse struct {
	BaseResponse
	Browse BrowseResult `json:"browse"`
}

// ShutdownRequest asks the controller to shut down ("shutdown"). MistServer
// only honours it over a local connection; Shutdown is the reason it logs.
type ShutdownRequest struct {
	authorizeRequest
	Shutdown string `json:"shutdown"`
}

// ShutdownResponse is "Shutting down", or "Ignored - only local users may
// request shutdown".
type ShutdownResponse struct {
	BaseResponse
	Shutdown string `json:"shutdown"`
}

type logoutRequest struct {
	authorizeRequest
	Logout bool `json:"logout"`
}

type clearStatLogsRequest struct {
	authorizeRequest
	ClearStatLogs bool `json:"clearstatlogs"`
}

type updateRequest struct {
	authorizeRequest
	Update bool `json:"update"`
}

type autoUpdateRequest struct {
	authorizeRequest
	AutoUpdate bool `json:"autoupdate"`
}

// UpdateInfo is MistServer's cached update status (refreshed about hourly).
// Only builds with the updater enabled send it.
type UpdateInfo struct {
	Error       string   `json:"error,omitempty"`
	Release     string   `json:"release"`
	Version     string   `json:"version"`
	Date        string   `json:"date"`
	UpToDate    int      `json:"uptodate"`     // 1 when up to date
	NeedsUpdate []string `json:"needs_update"` // not filled by current MistServer releases
	Progress    int      `json:"progress"`     // percent, only while updating
}

// UpdateResponse is the reply to PostUpdate and PostAutoUpdate.
type UpdateResponse struct {
	BaseResponse
	Update UpdateInfo `json:"update"`
}

func (s *service) PostConfig(request ConfigRequest) (*Response, error) {
	return doAuthorized[Response](s, request)
}

func (s *service) PostAddProtocol(request AddProtocolRequest) (*Response, error) {
	return doAuthorized[Response](s, request)
}

func (s *service) PostDeleteProtocol(request DeleteProtocolRequest) (*Response, error) {
	return doAuthorized[Response](s, request)
}

func (s *service) PostUpdateProtocol(request UpdateProtocolRequest) (*Response, error) {
	return doAuthorized[Response](s, updateProtocolWire{UpdateProtocol: [2]Protocol{request.Old, request.New}})
}

func (s *service) PostConfigBackup() (*ConfigBackupResponse, error) {
	return doAuthorized[ConfigBackupResponse](s, configBackupRequest{ConfigBackup: true})
}

func (s *service) PostConfigRestore(request ConfigRestoreRequest) error {
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(request.ConfigRestore, &cfg); err != nil || cfg == nil {
		return fmt.Errorf("%w: ConfigRestore must be a JSON object, or MistServer wipes its configuration", ErrInvalidRequest)
	}
	return doNoReply(s, request)
}

func (s *service) PostSave() error {
	return doNoReply(s, saveRequest{Save: true})
}

func (s *service) PostUISettings(request UISettingsRequest) (*UISettingsResponse, error) {
	wire := uiSettingsWire{UISettings: true}
	if request.UISettings != nil {
		wire.UISettings = request.UISettings
	}
	return doAuthorized[UISettingsResponse](s, wire)
}

func (s *service) PostAPIEndpoint() (*APIEndpointResponse, error) {
	return doAuthorized[APIEndpointResponse](s, apiEndpointRequest{APIEndpoint: true})
}

func (s *service) PostBrowse(request BrowseRequest) (*BrowseResponse, error) {
	return doAuthorized[BrowseResponse](s, request)
}

func (s *service) PostShutdown(request ShutdownRequest) (*ShutdownResponse, error) {
	return doAuthorized[ShutdownResponse](s, request)
}

func (s *service) PostLogout() error {
	return doNoReply(s, logoutRequest{Logout: true})
}

func (s *service) PostClearStatLogs() error {
	return doNoReply(s, clearStatLogsRequest{ClearStatLogs: true})
}

func (s *service) PostUpdate() (*UpdateResponse, error) {
	return doAuthorized[UpdateResponse](s, updateRequest{Update: true})
}

func (s *service) PostAutoUpdate() (*UpdateResponse, error) {
	return doAuthorized[UpdateResponse](s, autoUpdateRequest{AutoUpdate: true})
}
