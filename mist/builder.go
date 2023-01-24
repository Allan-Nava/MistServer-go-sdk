package mist_go

import (
	"log"

	"github.com/go-resty/resty/v2"
)

//
//
func BuildMist(url string, debug bool) (*MistGo, error) {
	mistClient := &MistGo{
		Url:        url,
		RestClient: resty.New(),
	}
	//
	if debug {
		mistClient.RestClient.SetDebug(true)
		mistClient.Debug = true
		log.Println("Debug mode is enabled for the Haproxy client ")
	}
	return mistClient, nil
}

//

func (o *MistGo) DebugPrint(data interface{}) {
	if o.Debug {
		log.Println(data)
	}
}
