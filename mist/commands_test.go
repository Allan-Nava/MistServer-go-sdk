package mist_go

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// commandCase checks one API method end to end against fakeMist: the exact
// command it puts on the wire (authorize stripped) and how it decodes a reply
// shaped like the one the MistServer controller sends.
type commandCase struct {
	name    string
	call    func(c IMistGoClient) (any, error)
	request string                    // expected command JSON, without "authorize"
	reply   string                    // fields merged into an authorized OK reply; "" for none
	check   func(t *testing.T, r any) // optional assertions on the decoded reply
}

func runCommandCases(t *testing.T, cases []commandCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reply := `{"authorize":{"status":"OK"}`
			if tc.reply != "" {
				reply += "," + strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(tc.reply), "{"), "}")
			}
			reply += "}"
			f := &fakeMist{username: "admin", password: "secret", challenge: "c1", reply: reply}
			c := newTestClient(t, f, "secret")

			resp, err := tc.call(c)
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			if len(f.commands) != 1 {
				t.Fatalf("authorized commands = %d, want 1", len(f.commands))
			}

			got := f.commands[0]
			delete(got, "authorize")
			var want map[string]any
			if err := json.Unmarshal([]byte(tc.request), &want); err != nil {
				t.Fatalf("bad want JSON: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				g, _ := json.Marshal(got)
				t.Errorf("request\n got: %s\nwant: %s", g, tc.request)
			}
			if tc.check != nil {
				tc.check(t, resp)
			}
		})
	}
}

// ---------------------------------------------------------------- streams

func TestStreamCommands(t *testing.T) {
	runCommandCases(t, []commandCase{
		{
			name: "streams replaces the stream list",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreams(StreamsRequest{Streams: map[string]AddStream{
					"live": {Name: "live", Source: "push://"},
				}})
			},
			request: `{"streams":{"live":{"name":"live","source":"push://"}}}`,
			reply:   `{"streams":{"live":{"name":"live","source":"push://","online":2}}}`,
			check: func(t *testing.T, r any) {
				if got := r.(*Response).Streams["live"].Online; got != 2 {
					t.Errorf("online = %d, want 2", got)
				}
			},
		},
		{
			name: "deletestreamsource reports per-stream status",
			call: func(c IMistGoClient) (any, error) {
				return c.PostDeleteStreamSource(DeleteStreamSourceRequest{DeleteStreamSource: []string{"vod"}})
			},
			request: `{"deletestreamsource":["vod"]}`,
			reply:   `{"deletestreamsource":["-2: Stream and source file deleted"]}`,
			check: func(t *testing.T, r any) {
				want := []string{"-2: Stream and source file deleted"}
				if got := r.(*DeleteStreamSourceResponse).DeleteStreamSource; !reflect.DeepEqual(got, want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name: "nuke_stream",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostNukeStream(NukeStreamRequest{NukeStream: "live"})
			},
			request: `{"nuke_stream":"live"}`,
		},
		{
			name: "no_unconfigured_streams",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostNoUnconfiguredStreams()
			},
			request: `{"no_unconfigured_streams":true}`,
		},
		{
			name: "tag_stream",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostTagStream(TagStreamRequest{TagStream: map[string][]string{"live": {"record"}}})
			},
			request: `{"tag_stream":{"live":["record"]}}`,
		},
		{
			name: "untag_stream",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostUntagStream(UntagStreamRequest{UntagStream: map[string][]string{"live": {"record"}}})
			},
			request: `{"untag_stream":{"live":["record"]}}`,
		},
		{
			name: "stream_tags for named streams",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreamTags(StreamTagsRequest{StreamTags: []string{"live", "vod"}})
			},
			request: `{"stream_tags":["live","vod"]}`,
			reply:   `{"stream_tags":{"live":["record"],"vod":null}}`,
			check: func(t *testing.T, r any) {
				got := r.(*StreamTagsResponse).StreamTags
				if !reflect.DeepEqual(got["live"], []string{"record"}) || got["vod"] != nil {
					t.Errorf("stream_tags = %v", got)
				}
			},
		},
		{
			name: "stream_tags with no names asks for all active streams",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreamTags(StreamTagsRequest{})
			},
			request: `{"stream_tags":true}`,
			reply:   `{"stream_tags":{}}`,
		},
	})
}

func TestStreamsRefusesNilMap(t *testing.T) {
	// {"streams": null} makes MistServer drop every configured stream.
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "secret")

	_, err := c.PostStreams(StreamsRequest{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
	if f.requests != 0 {
		t.Errorf("requests = %d, want 0 (nothing sent)", f.requests)
	}
}

// ---------------------------------------------------------------- stats

func TestStatsCommands(t *testing.T) {
	runCommandCases(t, []commandCase{
		{
			name: "active_streams always asks for the long form",
			call: func(c IMistGoClient) (any, error) {
				return c.PostActiveStreams(ActiveStreamsRequest{Fields: []string{"viewers", "lastms"}, Streams: []string{"live"}})
			},
			request: `{"active_streams":{"fields":["viewers","lastms"],"streams":["live"],"longform":true}}`,
			reply:   `{"active_streams":{"live":{"viewers":3,"lastms":120000}}}`,
			check: func(t *testing.T, r any) {
				if got := r.(*ActiveStreamsResponse).ActiveStreams["live"]["viewers"]; got != float64(3) {
					t.Errorf("viewers = %v, want 3", got)
				}
			},
		},
		{
			name: "active_streams with no filters",
			call: func(c IMistGoClient) (any, error) {
				return c.PostActiveStreams(ActiveStreamsRequest{})
			},
			request: `{"active_streams":{"longform":true}}`,
			reply:   `{"active_streams":null}`,
		},
		{
			name: "stats_streams names only",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStatsStreams(StatsStreamsRequest{})
			},
			request: `{"stats_streams":true}`,
			reply:   `{"stats_streams":["live","vod"]}`,
			check: func(t *testing.T, r any) {
				if got := r.(*StatsStreamsResponse).StatsStreams.Names; !reflect.DeepEqual(got, []string{"live", "vod"}) {
					t.Errorf("names = %v", got)
				}
			},
		},
		{
			name: "stats_streams with fields",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStatsStreams(StatsStreamsRequest{Fields: []string{"clients", "lastms"}})
			},
			request: `{"stats_streams":["clients","lastms"]}`,
			reply:   `{"stats_streams":{"live":[2,5000]}}`,
			check: func(t *testing.T, r any) {
				s := r.(*StatsStreamsResponse).StatsStreams
				if !reflect.DeepEqual(s.Values["live"], []any{float64(2), float64(5000)}) {
					t.Errorf("values = %v", s.Values)
				}
				if !reflect.DeepEqual(s.Names, []string{"live"}) {
					t.Errorf("names = %v, want [live]", s.Names)
				}
			},
		},
		{
			name: "clients",
			call: func(c IMistGoClient) (any, error) {
				return c.PostClients(ClientsRequest{Clients: ClientsQuery{Streams: []string{"live"}, Fields: []string{"host", "protocol"}, Time: -30}})
			},
			request: `{"clients":{"streams":["live"],"fields":["host","protocol"],"time":-30}}`,
			reply:   `{"clients":{"time":1790000000,"fields":["host","protocol"],"data":[["10.0.0.1","HLS"]]}}`,
			check: func(t *testing.T, r any) {
				cl := r.(*ClientsResponse).Clients
				if cl.Time != 1790000000 || len(cl.Data) != 1 || cl.Data[0][1] != "HLS" {
					t.Errorf("clients = %+v", cl)
				}
			},
		},
		{
			name: "totals",
			call: func(c IMistGoClient) (any, error) {
				return c.PostTotals(TotalsRequest{Totals: TotalsQuery{Fields: []string{"clients"}, Start: -60}})
			},
			request: `{"totals":{"fields":["clients"],"start":-60}}`,
			reply:   `{"totals":{"start":1,"end":61,"fields":["clients"],"interval":[[12,5]],"data":[[4]]}}`,
			check: func(t *testing.T, r any) {
				tt := r.(*TotalsResponse).Totals
				if tt.End != 61 || !reflect.DeepEqual(tt.Interval, [][]int64{{12, 5}}) {
					t.Errorf("totals = %+v", tt)
				}
			},
		},
		{
			name: "proc_list",
			call: func(c IMistGoClient) (any, error) {
				return c.PostProcList(ProcListRequest{ProcList: "live"})
			},
			request: `{"proc_list":"live"}`,
			reply:   `{"proc_list":{"4321":{"source":"live","sink":"live_720p","process":"AV","logs":[],"terminated":true}}}`,
			check: func(t *testing.T, r any) {
				p := r.(*ProcListResponse).ProcList["4321"]
				if p.Process != "AV" || !p.Terminated || p.Sink != "live_720p" {
					t.Errorf("proc = %+v", p)
				}
			},
		},
		{
			name: "capabilities",
			call: func(c IMistGoClient) (any, error) {
				return c.PostCapabilities()
			},
			request: `{"capabilities":true}`,
			reply: `{"capabilities":{"connectors":{"HLS":{"name":"HLS"}},"inputs":{"Buffer":{"name":"Buffer"}},
				"cpu_use":500,"cpu":[{"cores":4,"mhz":1645,"model":"x","threads":8}],
				"load":{"one":124,"five":81,"fifteen":72,"mem":42.7,"shm":3.1},
				"mem":{"total":7898,"used":3370,"free":2539,"cached":1989,"swaptotal":0,"swapfree":0},
				"speed":6580,"threads":8}}`,
			check: func(t *testing.T, r any) {
				cp := r.(*CapabilitiesResponse).Capabilities
				// controller_capabilities.cpp sends load.mem / load.shm as doubles;
				// the docs' "memory" key does not exist.
				if cp.Load.Mem != 42.7 || cp.Load.Shm != 3.1 {
					t.Errorf("load = %+v, want mem 42.7 shm 3.1", cp.Load)
				}
				if cp.CPUUse != 500 || cp.Mem.Total != 7898 || cp.Load.One != 124 || len(cp.CPU) != 1 {
					t.Errorf("capabilities = %+v", cp)
				}
				if _, ok := cp.Connectors["HLS"]; !ok {
					t.Errorf("connectors = %v", cp.Connectors)
				}
			},
		},
	})
}

// ---------------------------------------------------------------- push

func TestPushCommands(t *testing.T) {
	runCommandCases(t, []commandCase{
		{
			name: "push_start",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostPushStart(PushStartRequest{PushStart: PushStart{Stream: "live", Target: "rtmp://x/app/KEY"}})
			},
			request: `{"push_start":{"stream":"live","target":"rtmp://x/app/KEY"}}`,
		},
		{
			name: "push_auto_list",
			call: func(c IMistGoClient) (any, error) {
				return c.PostPushAutoList()
			},
			request: `{"push_auto_list":true}`,
			reply: `{"auto_push":{"71d6":{"stream":"live","target":"/tmp/$stream.mkv","x-LSP-notes":"n",
				"scheduletime":100,"start_rule":["hour",11,9]}}}`,
			check: func(t *testing.T, r any) {
				p := r.(*PushAutoListResponse).AutoPush["71d6"]
				if p.Stream != "live" || p.Notes != "n" || p.ScheduleTime != 100 || len(p.StartRule) != 3 {
					t.Errorf("auto push = %+v", p)
				}
			},
		},
		{
			name: "push_settings only sends what is set",
			call: func(c IMistGoClient) (any, error) {
				wait := 5
				return c.PostPushSettings(PushSettingsRequest{PushSettings: PushSettingsUpdate{Wait: &wait}})
			},
			request: `{"push_settings":{"wait":5}}`,
			reply:   `{"push_settings":{"wait":5,"maxspeed":0}}`,
			check: func(t *testing.T, r any) {
				if got := r.(*PushSettingsResponse).PushSettings; got.Wait != 5 || got.MaxSpeed != 0 {
					t.Errorf("settings = %+v", got)
				}
			},
		},
		{
			name: "push_settings read-only",
			call: func(c IMistGoClient) (any, error) {
				return c.PostPushSettings(PushSettingsRequest{})
			},
			request: `{"push_settings":{}}`,
			reply:   `{"push_settings":{"wait":0,"maxspeed":0}}`,
		},
	})
}

// ---------------------------------------------------------------- sessions

func TestSessionCommands(t *testing.T) {
	runCommandCases(t, []commandCase{
		{
			name: "stop_sessions uses the stream-to-protocol object form",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostStopSessions(StopSessionsRequest{StopSessions: map[string]string{"live": "", "": "RTMP"}})
			},
			request: `{"stop_sessions":{"live":"","":"RTMP"}}`,
		},
		{
			name: "stop_sessid",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostStopSessID(StopSessIDRequest{StopSessID: []string{"abc"}})
			},
			request: `{"stop_sessid":["abc"]}`,
		},
		{
			name: "stop_tag",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostStopTag(StopTagRequest{StopTag: []string{"banned"}})
			},
			request: `{"stop_tag":["banned"]}`,
		},
		{
			name: "tag_sessid",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostTagSessID(TagSessIDRequest{TagSessID: map[string]string{"abc": "vip"}})
			},
			request: `{"tag_sessid":{"abc":"vip"}}`,
		},
		{
			name: "invalidate_sessions",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostInvalidateSessions(InvalidateSessionsRequest{InvalidateSessions: []string{"live"}})
			},
			request: `{"invalidate_sessions":["live"]}`,
		},
	})
}

// ---------------------------------------------------------------- config

func TestConfigCommands(t *testing.T) {
	runCommandCases(t, []commandCase{
		{
			name: "config",
			call: func(c IMistGoClient) (any, error) {
				return c.PostConfig(ConfigRequest{Config: map[string]any{"serverid": "edge-1"}})
			},
			request: `{"config":{"serverid":"edge-1"}}`,
			reply:   `{"config":{"serverid":"edge-1","version":"3.4"}}`,
			check: func(t *testing.T, r any) {
				if got := r.(*Response).Config.Version; got != "3.4" {
					t.Errorf("version = %q", got)
				}
			},
		},
		{
			name: "addprotocol",
			call: func(c IMistGoClient) (any, error) {
				return c.PostAddProtocol(AddProtocolRequest{AddProtocol: []Protocol{{"connector": "RTMP", "port": 1935}}})
			},
			request: `{"addprotocol":[{"connector":"RTMP","port":1935}]}`,
		},
		{
			name: "deleteprotocol",
			call: func(c IMistGoClient) (any, error) {
				return c.PostDeleteProtocol(DeleteProtocolRequest{DeleteProtocol: []Protocol{{"connector": "RTMP"}}})
			},
			request: `{"deleteprotocol":[{"connector":"RTMP"}]}`,
		},
		{
			name: "updateprotocol sends [old, new]",
			call: func(c IMistGoClient) (any, error) {
				return c.PostUpdateProtocol(UpdateProtocolRequest{
					Old: Protocol{"connector": "RTMP"},
					New: Protocol{"connector": "RTMP", "port": 1936},
				})
			},
			request: `{"updateprotocol":[{"connector":"RTMP"},{"connector":"RTMP","port":1936}]}`,
		},
		{
			name: "config_backup",
			call: func(c IMistGoClient) (any, error) {
				return c.PostConfigBackup()
			},
			request: `{"config_backup":true}`,
			reply:   `{"config_backup":{"streams":{}}}`,
			check: func(t *testing.T, r any) {
				if got := string(r.(*ConfigBackupResponse).ConfigBackup); got != `{"streams":{}}` {
					t.Errorf("backup = %s", got)
				}
			},
		},
		{
			name: "config_restore",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostConfigRestore(ConfigRestoreRequest{ConfigRestore: json.RawMessage(`{"streams":{}}`)})
			},
			request: `{"config_restore":{"streams":{}}}`,
		},
		{
			name: "save",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostSave()
			},
			request: `{"save":true}`,
		},
		{
			name: "ui_settings store",
			call: func(c IMistGoClient) (any, error) {
				return c.PostUISettings(UISettingsRequest{UISettings: map[string]any{"theme": "dark"}})
			},
			request: `{"ui_settings":{"theme":"dark"}}`,
			reply:   `{"ui_settings":{"theme":"dark"}}`,
		},
		{
			name: "ui_settings read",
			call: func(c IMistGoClient) (any, error) {
				return c.PostUISettings(UISettingsRequest{})
			},
			request: `{"ui_settings":true}`,
			reply:   `{"ui_settings":{"theme":"dark"}}`,
			check: func(t *testing.T, r any) {
				if got := string(r.(*UISettingsResponse).UISettings); got != `{"theme":"dark"}` {
					t.Errorf("ui_settings = %s", got)
				}
			},
		},
		{
			name: "api_endpoint",
			call: func(c IMistGoClient) (any, error) {
				return c.PostAPIEndpoint()
			},
			request: `{"api_endpoint":true}`,
			reply:   `{"api_endpoint":"http://127.0.0.1:4242/"}`,
			check: func(t *testing.T, r any) {
				if got := r.(*APIEndpointResponse).APIEndpoint; got != "http://127.0.0.1:4242/" {
					t.Errorf("api_endpoint = %q", got)
				}
			},
		},
		{
			name: "browse takes a plain path string",
			call: func(c IMistGoClient) (any, error) {
				return c.PostBrowse(BrowseRequest{Browse: "/media"})
			},
			request: `{"browse":"/media"}`,
			reply:   `{"browse":{"path":["/media"],"files":["a.mp4"],"subdirectories":["old"]}}`,
			check: func(t *testing.T, r any) {
				b := r.(*BrowseResponse).Browse
				if b.Dir() != "/media" || !reflect.DeepEqual(b.Files, []string{"a.mp4"}) || !reflect.DeepEqual(b.Subdirectories, []string{"old"}) {
					t.Errorf("browse = %+v", b)
				}
			},
		},
		{
			name: "shutdown",
			call: func(c IMistGoClient) (any, error) {
				return c.PostShutdown(ShutdownRequest{Shutdown: "maintenance"})
			},
			request: `{"shutdown":"maintenance"}`,
			reply:   `{"shutdown":"Ignored - only local users may request shutdown"}`,
			check: func(t *testing.T, r any) {
				if got := r.(*ShutdownResponse).Shutdown; !strings.HasPrefix(got, "Ignored") {
					t.Errorf("shutdown = %q", got)
				}
			},
		},
		{
			name: "logout",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostLogout()
			},
			request: `{"logout":true}`,
		},
		{
			name: "clearstatlogs uses the key the server reads",
			call: func(c IMistGoClient) (any, error) {
				return nil, c.PostClearStatLogs()
			},
			request: `{"clearstatlogs":true}`,
		},
		{
			name: "update",
			call: func(c IMistGoClient) (any, error) {
				return c.PostUpdate()
			},
			request: `{"update":true}`,
			reply:   `{"update":{"release":"Generic_64","version":"3.5","date":"Jan 1","uptodate":0,"needs_update":["MistController"],"progress":10}}`,
			check: func(t *testing.T, r any) {
				u := r.(*UpdateResponse).Update
				if u.UpToDate != 0 || u.Progress != 10 || !reflect.DeepEqual(u.NeedsUpdate, []string{"MistController"}) {
					t.Errorf("update = %+v", u)
				}
			},
		},
		{
			name: "autoupdate",
			call: func(c IMistGoClient) (any, error) {
				return c.PostAutoUpdate()
			},
			request: `{"autoupdate":true}`,
			reply:   `{"update":{"uptodate":1}}`,
		},
	})
}

// ---------------------------------------------------------------- security

func TestSecurityCommands(t *testing.T) {
	jwk := JWK{
		Key:         map[string]any{"kty": "oct", "alg": "HS256", "k": "KEY", "kid": "k1"},
		Permissions: &JWKPermissions{Input: true, Output: true, Stream: "*"},
	}
	runCommandCases(t, []commandCase{
		{
			name: "streamkeys read",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreamKeys(StreamKeysRequest{})
			},
			request: `{"streamkeys":true}`,
			reply:   `{"streamkeys":{"abc":"live"}}`,
			check: func(t *testing.T, r any) {
				if got := r.(*StreamKeysResponse).StreamKeys["abc"]; got != "live" {
					t.Errorf("streamkeys = %v", r)
				}
			},
		},
		{
			name: "streamkeys replace",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreamKeys(StreamKeysRequest{StreamKeys: map[string]string{"abc": "live"}})
			},
			request: `{"streamkeys":{"abc":"live"}}`,
			reply:   `{"streamkeys":{"abc":"live"}}`,
		},
		{
			name: "streamkey_add",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreamKeyAdd(StreamKeyAddRequest{StreamKeyAdd: map[string]string{"abc": "live"}})
			},
			request: `{"streamkey_add":{"abc":"live"}}`,
			reply:   `{"streamkey_add":{"added":["abc"]}}`,
			check: func(t *testing.T, r any) {
				if got := r.(*StreamKeyAddResponse).StreamKeyAdd.Added; !reflect.DeepEqual(got, []string{"abc"}) {
					t.Errorf("added = %v", got)
				}
			},
		},
		{
			name: "streamkey_del",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreamKeyDel(StreamKeyDelRequest{StreamKeyDel: []string{"abc"}})
			},
			request: `{"streamkey_del":["abc"]}`,
			reply:   `{"streamkey_del":{"deleted":["abc"]}}`,
			check: func(t *testing.T, r any) {
				if got := r.(*StreamKeyDelResponse).StreamKeyDel.Deleted; !reflect.DeepEqual(got, []string{"abc"}) {
					t.Errorf("deleted = %v", got)
				}
			},
		},
		{
			name: "jwks read",
			call: func(c IMistGoClient) (any, error) {
				return c.PostJWKS(JWKSRequest{})
			},
			request: `{"jwks":true}`,
			reply:   `{"jwks":[[{"kid":"k1"},{"input":true,"output":true,"admin":false,"stream":"*"}]]}`,
			check: func(t *testing.T, r any) {
				if got := r.(*JWKSResponse).JWKS; len(got) != 1 {
					t.Errorf("jwks = %s", got)
				}
			},
		},
		{
			name: "jwks replace, key with permissions is sent as a pair",
			call: func(c IMistGoClient) (any, error) {
				return c.PostJWKS(JWKSRequest{JWKS: []JWK{jwk}})
			},
			request: `{"jwks":[[{"kty":"oct","alg":"HS256","k":"KEY","kid":"k1"},{"input":true,"output":true,"admin":false,"stream":"*"}]]}`,
			reply:   `{"jwks":[]}`,
		},
		{
			name: "addjwks, a bare URL is sent as a string",
			call: func(c IMistGoClient) (any, error) {
				return c.PostAddJWKS(AddJWKSRequest{AddJWKS: []JWK{{Key: "https://issuer.example.com/jwks.json"}}})
			},
			request: `{"addjwks":["https://issuer.example.com/jwks.json"]}`,
			reply:   `{"addjwks":["https://issuer.example.com/jwks.json"]}`,
		},
		{
			name: "deletejwks by key id",
			call: func(c IMistGoClient) (any, error) {
				return c.PostDeleteJWKS(DeleteJWKSRequest{DeleteJWKS: []JWK{{Key: map[string]any{"kid": "k1"}}}})
			},
			request: `{"deletejwks":[{"kid":"k1"}]}`,
			reply:   `{"deletejwks":[{"kid":"k1"}]}`,
		},
	})
}

// ---------------------------------------------------------------- variables & external writers

func TestVariableCommands(t *testing.T) {
	varsReply := `{"variable_list":{"host":{"value":"edge-1"},"load":{"target":"cat /proc/loadavg","interval":9.5,"waitTime":1}}}`
	checkVars := func(t *testing.T, r any) {
		v := r.(*VariablesResponse).VariableList
		if v["host"].Value != "edge-1" || v["load"].Interval != 9.5 || v["load"].WaitTime != 1 {
			t.Errorf("variables = %+v", v)
		}
	}
	writer := ExternalWriter{Name: "s3", CmdLine: "/usr/bin/s3up", Protocols: []string{"s3"}}
	runCommandCases(t, []commandCase{
		{
			name:    "variable_list",
			call:    func(c IMistGoClient) (any, error) { return c.PostVariableList() },
			request: `{"variable_list":true}`,
			reply:   varsReply,
			check:   checkVars,
		},
		{
			name: "variable_add object form",
			call: func(c IMistGoClient) (any, error) {
				return c.PostVariableAdd(VariableAddRequest{VariableAdd: Variable{Name: "host", Value: "edge-1"}})
			},
			request: `{"variable_add":{"name":"host","value":"edge-1"}}`,
			reply:   varsReply,
			check:   checkVars,
		},
		{
			name: "variable_remove",
			call: func(c IMistGoClient) (any, error) {
				return c.PostVariableRemove(VariableRemoveRequest{VariableRemove: []string{"host"}})
			},
			request: `{"variable_remove":["host"]}`,
			reply:   varsReply,
		},
		{
			name:    "external_writer_list decodes [name, cmdline, protocols] rows",
			call:    func(c IMistGoClient) (any, error) { return c.PostExternalWriterList() },
			request: `{"external_writer_list":true}`,
			reply:   `{"external_writer_list":[["s3","/usr/bin/s3up",["s3"]]]}`,
			check: func(t *testing.T, r any) {
				got := r.(*ExternalWritersResponse).ExternalWriterList
				if len(got) != 1 || !reflect.DeepEqual(got[0], writer) {
					t.Errorf("writers = %+v", got)
				}
			},
		},
		{
			name: "external_writer_add sends the object form",
			call: func(c IMistGoClient) (any, error) {
				return c.PostExternalWriterAdd(ExternalWriterAddRequest{ExternalWriterAdd: writer})
			},
			request: `{"external_writer_add":{"name":"s3","cmdline":"/usr/bin/s3up","protocols":["s3"]}}`,
			reply:   `{"external_writer_list":[["s3","/usr/bin/s3up",["s3"]]]}`,
		},
		{
			name: "external_writer_remove",
			call: func(c IMistGoClient) (any, error) {
				return c.PostExternalWriterRemove(ExternalWriterRemoveRequest{ExternalWriterRemove: "s3"})
			},
			request: `{"external_writer_remove":"s3"}`,
			reply:   `{"external_writer_list":[]}`,
		},
	})
}

// ---------------------------------------------------------------- raw

func TestRawCommand(t *testing.T) {
	runCommandCases(t, []commandCase{
		{
			name: "combines commands and returns every reply member",
			call: func(c IMistGoClient) (any, error) {
				return c.PostRaw(map[string]any{"push_stop_graceful": []int{412}, "api_endpoint": true})
			},
			request: `{"push_stop_graceful":[412],"api_endpoint":true}`,
			reply:   `{"api_endpoint":"http://127.0.0.1:4242/"}`,
			check: func(t *testing.T, r any) {
				got := r.(map[string]json.RawMessage)
				if string(got["api_endpoint"]) != `"http://127.0.0.1:4242/"` {
					t.Errorf("api_endpoint = %s", got["api_endpoint"])
				}
			},
		},
	})
}

func TestRawCommandChecksAuthorization(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "wrong")

	if _, err := c.PostRaw(map[string]any{"api_endpoint": true}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestRawCommandRejectsAuthorizeKey(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "secret")

	_, err := c.PostRaw(map[string]any{"authorize": map[string]any{"username": "x"}})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

// ---------------------------------------------------------------- original commands

func TestOriginalCommands(t *testing.T) {
	runCommandCases(t, []commandCase{
		{
			name: "deletestream",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreamRemove(PostStreamRemoveRequest{DeleteStream: "live"})
			},
			request: `{"deletestream":"live"}`,
		},
		{
			name: "push_auto_add",
			call: func(c IMistGoClient) (any, error) {
				return c.PostAutoPush(PostAutoPushRequest{PushAutoAdd: PushAutoAdd{Stream: "live", Target: "rtmp://x/app/KEY"}})
			},
			request: `{"push_auto_add":{"stream":"live","target":"rtmp://x/app/KEY"}}`,
		},
		{
			name: "push_auto_remove",
			call: func(c IMistGoClient) (any, error) {
				return c.PostAutoPushRemove(PostAutoPushRemoveRequest{PushAutoRemove: "live"})
			},
			request: `{"push_auto_remove":"live"}`,
		},
		{
			name: "push_stop",
			call: func(c IMistGoClient) (any, error) {
				return c.PostPushStop(PostPushStopRequest{PushStop: []int{412}})
			},
			request: `{"push_stop":[412]}`,
		},
	})
}

// ---------------------------------------------------------------- audit fixes

func TestStreamWritesSendTheFullStreamObject(t *testing.T) {
	// addstream/streams replace a stream's whole config (AddStreams in
	// controller_streams.cpp), so callers must be able to send every setting,
	// not only the typed ones. stop_sessions is only read as a top-level
	// command, so the SDK lifts it out of the stream.
	live := AddStream{
		Name: "live", Source: "push://", DVR: 30000, StopSessions: true,
		Options: map[string]any{"always_on": true, "tags": []string{"record"}},
	}
	runCommandCases(t, []commandCase{
		{
			name: "addstream",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStream(PostStreamRequest{AddStream: map[string]AddStream{"live": live}})
			},
			request: `{"addstream":{"live":{"name":"live","source":"push://","DVR":30000,"always_on":true,"tags":["record"]}},
				"stop_sessions":{"live":""}}`,
		},
		{
			name: "streams",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStreams(StreamsRequest{Streams: map[string]AddStream{"live": live}})
			},
			request: `{"streams":{"live":{"name":"live","source":"push://","DVR":30000,"always_on":true,"tags":["record"]}},
				"stop_sessions":{"live":""}}`,
		},
		{
			name: "typed fields win over Options",
			call: func(c IMistGoClient) (any, error) {
				return c.PostStream(PostStreamRequest{AddStream: map[string]AddStream{
					"live": {Name: "live", Source: "push://", Options: map[string]any{"source": "ignored"}},
				}})
			},
			request: `{"addstream":{"live":{"name":"live","source":"push://"}}}`,
		},
	})
}

func TestConfigRestoreRefusesEmptyOrNonObject(t *testing.T) {
	// config_restore replaces the entire server state (Storage.assignFrom); a
	// null or non-object value would wipe accounts, config and keys.
	for _, cfg := range []json.RawMessage{nil, json.RawMessage(``), json.RawMessage(`null`), json.RawMessage(`[1]`)} {
		f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
		c := newTestClient(t, f, "secret")

		err := c.PostConfigRestore(ConfigRestoreRequest{ConfigRestore: cfg})
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("config %q: err = %v, want ErrInvalidRequest", cfg, err)
		}
		if f.requests != 0 {
			t.Errorf("config %q: requests = %d, want 0", cfg, f.requests)
		}
	}
}
