package mist_go

/*
	{
	    "authorize": {
	        "challenge": "",
	        "status": "CHALL"
	    }
	}
*/
type ResponseBase struct {
	Authorize Authorize `json:"authorize"`
}

type Authorize struct {
	Challenge string `json:"challenge"`
	Status    string `json:"status"`
}
