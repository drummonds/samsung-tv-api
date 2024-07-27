package main

import (
	"github.com/avbdr/samsung-tv-api/pkg/device"
	samsung_tv_api "github.com/avbdr/samsung-tv-api/pkg/samsung-tv-api"
	sonos_api "github.com/avbdr/samsung-tv-api/pkg/sonos-api"
	//"github.com/davecgh/go-spew/spew"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

var devices_ []device.DeviceInfo

func zeroconfDisco() {
	samsungs := samsung_tv_api.Discover()
	for _, dev := range samsungs {
		if !device.Exists(devices_, dev) {
			log.Printf("Found %v\n", dev)
			devices_ = append(devices_, dev)
		}
	}

	sonosDevs := sonos_api.Discover()
	for _, dev := range sonosDevs {
		if !device.Exists(devices_, dev) {
			log.Printf("Found %v\n", dev)
			devices_ = append(devices_, dev)
		}
	}
}

func saveConfig() {
	if devices_ == nil {
		return
	}
	configBytes, err := json.MarshalIndent(devices_, "", "  ")
	if err != nil {
		fmt.Printf("1 %v", err)
	}
	homeDir, _ := os.UserHomeDir()
	err = ioutil.WriteFile(homeDir+"/.samsung.json", configBytes, 0644)
	if err != nil {
		fmt.Printf("2 %v", err)
	}

}

func loadConfig() {
	homeDir, _ := os.UserHomeDir()
	configData, err := ioutil.ReadFile(homeDir + "/.samsung.json")
	if err != nil {
		return
	}
	err = json.Unmarshal(configData, &devices_)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	deviceId := 0
	var help bool
	flag.IntVar(&deviceId, "d", 0, "Speaker id is not defined")
	flag.BoolVar(&help, "help", false, "Help")
	flag.Parse()
	Args := flag.Args()

	if len(os.Args) == 1 {
		help = true
	}
	if help {
		flag.PrintDefaults()
		return
	}

	loadConfig()
	/*
		if len(os.Args) == 3 {
			deviceId = 0 // fixme. find the right tv from the config by name
			fmt.Printf("TV: %s", os.Args[2])
		}
	*/

	if Args[0] == "devices" {
		for id, d := range devices_ {
			fmt.Printf("%d - %s: %s - %s\n", id, d.Type, d.Name, d.Ip)
		}
		return
	}
	if Args[0] == "discover" {
		zeroconfDisco()
		saveConfig()
		return
	}

	var tv device.DeviceInfo
	tv = devices_[deviceId]
	log.Printf("Device: %#v", tv)
	var devApi device.Device
	if tv.Type == "samsungtv" {
		devApi = samsung_tv_api.NewSamsungTvWebSocket(&devices_[deviceId], 0, false)
		devApi.Init()
		saveConfig()
	} else if tv.Type == "sonos" {
		devApi = sonos_api.NewSonosDevice(tv.Ip)
	} else {
		log.Printf("Error: unsupported device type: %s", tv.Type)
		return
	}

	if Args[0] == "poweroff" {
		devApi.PowerOff()
		return
	}

	if Args[0] == "list" {
		devApi.List()
		return
	}

	if Args[0] == "open" {
		if flag.NArg() != 2 {
			log.Fatal("no url or app specified")
		}
		devApi.Open(Args[1])
		return
	}
	if Args[0] == "key" {
		if flag.NArg() != 2 {
			log.Fatal("no key specified")
		}
		devApi.Key(Args[1])
		return
	}
	if Args[0] == "volup" {
		devApi.VolUp()
		return
	}

	if Args[0] == "voldown" {
		devApi.VolDown()
		return
	}

	if Args[0] == "vol" {
		log.Printf("%d", flag.NArg())
		if flag.NArg() != 2 {
			vol, _ := devApi.Vol(-1)
			log.Printf("volume is %d", vol)
			return
		}
		intValue, _ := strconv.Atoi(Args[1])
		devApi.Vol(intValue)
		return
	}

	if Args[0] == "test" {
		devApi.Test()
	}

	if Args[0] == "text" {
		if flag.NArg() != 2 {
			log.Fatal("no key specified")
		}
		devApi.Text(Args[1])
		return
	}

	if Args[0] == "stream" {
		if len(os.Args) != 3 {
			log.Fatal("no key specified")
		}
		log.Printf("Streaming %s", os.Args[2])
		devApi.Stream(os.Args[2])
		return
	}

	if Args[0] == "status" {
		devApi.Status()
		return
	}

	if Args[0] == "next" {
		devApi.Next()
	}
	if Args[0] == "prev" {
		devApi.Prev()
	}
	if Args[0] == "pause" {
		devApi.Pause()
	}
	if Args[0] == "play" {
		devApi.Play()
	}
	if Args[0] == "status" {
		devApi.Status()
	}
}
