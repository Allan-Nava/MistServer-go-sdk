package mist_go

type AuthorizationResponse struct {
	Authorize Authorize `json:"authorize"`
}

type BaseResponse struct {
	LTS       int             `json:"LTS"`
	Authorize Authorize       `json:"authorize"`
	Config    Config          `json:"config"`
	Log       [][]interface{} `json:"log"`
}

type Response struct {
	BaseResponse
	Streams map[string]Stream `json:"streams"`
}

type PostStreamResponse struct {
	BaseResponse
	Streams any `json:"streams,omitempty"`
}

type PostPushListResponse struct {
	BaseResponse
	PushList [][]any `json:"push_list"`
}

type Authorize struct {
	Challenge string `json:"challenge"`
	Status    string `json:"status"`
}

type Stream struct {
	Debug        int           `json:"debug"`
	Name         string        `json:"name"`
	Online       int           `json:"online"`
	Processes    []interface{} `json:"processes"`
	Source       string        `json:"source"`
	StopSessions bool          `json:"stop_sessions"`
	Tags         []interface{} `json:"tags"`
}

type Config struct {
	Accesslog  string `json:"accesslog"`
	Controller struct {
		Interface interface{} `json:"interface"`
		Port      interface{} `json:"port"`
		Username  interface{} `json:"username"`
	} `json:"controller"`
	Debug         interface{} `json:"debug"`
	DefaultStream interface{} `json:"defaultStream"`
	Iid           string      `json:"iid"`
	Limits        interface{} `json:"limits"`
	Location      struct {
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
		Name string  `json:"name"`
	} `json:"location"`
	Prometheus string `json:"prometheus"`
	Protocols  []struct {
		Connector  string      `json:"connector"`
		Online     interface{} `json:"online"`
		Acceptable string      `json:"acceptable,omitempty"`
	} `json:"protocols"`
	Serverid               interface{} `json:"serverid"`
	SessionInputMode       int         `json:"sessionInputMode"`
	SessionOutputMode      int         `json:"sessionOutputMode"`
	SessionStreamInfoMode  int         `json:"sessionStreamInfoMode"`
	SessionUnspecifiedMode int         `json:"sessionUnspecifiedMode"`
	SessionViewerMode      int         `json:"sessionViewerMode"`
	Time                   int         `json:"time"`
	TknMode                int         `json:"tknMode"`
	Triggers               interface{} `json:"triggers"`
	Trustedproxy           []string    `json:"trustedproxy"`
	Version                string      `json:"version"`
}
