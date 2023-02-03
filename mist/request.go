package mist_go

/*
	{"authorize":{"username":"","password":""}}
*/
type AuthCommand struct {
	Authorize AuthorizeCommand `json:"authorize"`
}

type AuthorizeCommand struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
//

/*
{"capabilities": true }
*/
type CapabilitiesCommand struct {
	Capabilites bool `json:"capabilities"`
}

/*
{"addstream": {"ali": {"source": "dtsc://192.168.1.52:4200/live"}}}
*/
type AddStreamCommand struct {
	AddStream map[string]interface{} `json:"addstream"`
}

/*
{"active_streams":""}
*/
type ActiveStreamsCommand struct {
	ActiveStreams string `json:"active_streams"`
}

/*
{"push_start":{"stream": "","target": "rtmp://",}}
*/
type AddPushCommand struct {
	PushStart PushStartCommand `json:"push_start"`
}
type PushStartCommand struct {
	Stream string `json:"stream"`
	Target string `json:"target"`
}

/*
{"push_list":true}
*/
type PushListCommand struct {
	PushList bool `json:"push_list"`
}

/*
{
	"active_streams": [
		"clients",
		"lastms",
		"firstms",
		"viewers",
		"inputs",
		"outputs",
		"views",
		"viewseconds",
		"upbytes", 
		"downbytes", 
		"packsent",
		"packloss",
		"packretrans",
		"zerounix",
		"health", 
		"tracks", 
		"status"
	]
}
*/