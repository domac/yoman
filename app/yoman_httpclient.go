package yoman

import (
	client "github.com/domac/yoman/httpclient"
)

var yomanClient *client.HttpClient

const (
	TIMEOUT         = 5
	CONNECT_TIMEOUT = 5
)

func init() {
	yomanClient = client.NewHttpClient()
	yomanClient.Defaults(client.Map{
		"opt_timeout":        TIMEOUT,
		"opt_connecttimeout": CONNECT_TIMEOUT,
	})
}
