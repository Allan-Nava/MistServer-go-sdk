package mist_go

type postAuthorizeRequest struct{}
type healthRequest struct {
	authorizeRequest
}

type PostStreamRequest struct {
	authorizeRequest
	AddStream map[string]AddStream `json:"addstream"`
}

type AddStream struct {
	Name         string `json:"name"`
	Source       string `json:"source"`
	StopSessions bool   `json:"stop_sessions"`
	DVR          int    `json:"DVR"`
	Debug        int    `json:"debug"`
}

type PostAutoPushRequest struct {
	authorizeRequest
	PushAutoAdd PushAutoAdd `json:"push_auto_add"`
}

type PushAutoAdd struct {
	Stream string `json:"stream"`
	Target string `json:"target"`
}

type PostAutoPushStopRequest struct {
	authorizeRequest
	PushAutoRemove string `json:"push_auto_remove"`
}

type PostPushListRequest struct {
	authorizeRequest
	PushList bool `json:"push_list"`
}

type PostPushStopRequest struct {
	authorizeRequest
	PushStop []int `json:"push_stop"`
}

type PostAutoPushRemoveRequest struct {
	authorizeRequest
	PushAutoRemove string `json:"push_auto_remove"`
}

type PostStreamRemoveRequest struct {
	authorizeRequest
	DeleteStream string `json:"deletestream"`
}

type authorizeRequest struct {
	Authorize authorizeRequestInner `json:"authorize"`
}

type authorizeRequestInner struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
