package main

import (
	"github.com/avbdr/samsung-tv-api/internal/app/samsung-tv-api/helpers"
	samsung_tv_api "github.com/avbdr/samsung-tv-api/pkg/samsung-tv-api"
	"log"
)

func main() {
	config := helpers.LoadConfiguration()

	c := samsung_tv_api.NewSamsungTvWebSocket(
		"192.168.1.188",
		config.Token,
		8002,
		500,
		"stephenLaptop",
		true)

	device, err := c.Rest.GetDeviceInfo()

	if err != nil {
		log.Fatalln(err)
	}

	updatedToken := c.GetToken()

	if updatedToken != "" && updatedToken != config.Token {
		config.Token = updatedToken
	}

	config.Mac = device.Device.WifiMac
	_ = helpers.SaveConfiguration(&config)

	_ = c.Disconnect()
}
