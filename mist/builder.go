package mist_go

import (
	"log"

	"github.com/go-resty/resty/v2"
)

//
//
func BuildMist(url string, debug bool) (*MistGo, error) {
	mistClient := &MistGo{
		debug: 		debug,
		Url:        url,
		restClient: resty.New(),
	}
	// You can override all below settings and options at request level if you want to
	//--------------------------------------------------------------------------------
	// Host URL for all request. So you can use relative URL in the request
	mistClient.restClient.SetHostURL(url)
	// Headers for all request
	//mistClient.restClient.SetHeader("", "Basic ")
	//
	if debug {
		mistClient.restClient.SetDebug(true)
		mistClient.debug = true
		log.Println("Debug mode is enabled for the mistClient client ")
	}
	return mistClient, nil
}

//

func (o *MistGo) debugPrint(data interface{}) {
	if o.Debug {
		log.Println(data)
	}
}
