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
	AddStream map[string]interface{}
}