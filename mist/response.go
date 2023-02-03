package mist_go


/*
{
    "authorize": {
        "challenge": "",
        "status": "CHALL"
    }
}
*/
type ResponseAuth struct {
	Authorize Authorize `json:"authorize"`
}

type Authorize struct {
	Challenge 	string `json:"challenge"`
	Status 		string `json:"status"`
}