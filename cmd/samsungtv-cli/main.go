package main

import (
	"errors"

	"github.com/stephensli/samsung-tv-api/pkg/device"
	samsung_tv_api "github.com/stephensli/samsung-tv-api/pkg/samsung-tv-api"
	sonos_api "github.com/stephensli/samsung-tv-api/pkg/sonos-api"

	//"github.com/davecgh/go-spew/spew"
	"encoding/json"
	"flag"
	"fmt"
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
	err = os.WriteFile(homeDir+"/.samsung.json", configBytes, 0644)
	if err != nil {
		fmt.Printf("2 %v", err)
	}

}

func loadConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("user's home directory problem %v", err))
	}
	filename := homeDir + "/.samsung.json"
	_, err = os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			devices_ = make([]device.DeviceInfo, 0)
			return //
		}
		panic(fmt.Errorf("Something odd about file existence %s %v", filename, err))
	}
	configData, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Errorf("\nproblem reading file\n %v", err))
	}
	err = json.Unmarshal(configData, &devices_)
	if err != nil {
		panic(fmt.Errorf("\n~/.samsung.json is corrupt\n %v", err))
	}
}

const _usage = `Sub commands
DEVICE CONTROL
devices  Does a scan on the local network to find devices
ip  eg samsung-tv-api ip 192.168.1.2   Creates a device record for IP address
discover
COMMANDS
poweroff
list
open
key keynumber
volup
voldown
vol value
test
text
stream
status
next
prev
pause
play
status
`

func setUpFlag() {
	flag.ErrHelp = errors.New("flag: help requested")
	flag.Usage = func() {
		fmt.Println("Start of help 4")
		fmt.Fprint(flag.CommandLine.Output(), "Usage of samsungtv-cli:\n")
		fmt.Fprint(flag.CommandLine.Output(), _usage)
		flag.PrintDefaults()
	}
	flag.Parse()

}

func main() {
	deviceId := 0
	flag.IntVar(&deviceId, "d", 0, "Device or speaker id is not defined, 0 default")
	setUpFlag()

	Args := flag.Args()

	if len(Args) < 1 {
		flag.Usage()
		// flag.PrintDefaults()
		return
	}

	loadConfig()
	if Args[0] == "devices" {
		for id, d := range devices_ {
			fmt.Printf("%d - %s: %s - %s\n", id, d.Type, d.Name, d.Ip)
		}
		return
	}
	if Args[0] == "ip" {
		// read config
		ipAddress := Args[1]
		thisIp := func(d device.DeviceInfo) bool { return d.Ip != ipAddress }
		devices_ = filter(devices_, thisIp)
		d := setupDevice(ipAddress)
		devices_ = append(devices_, d)
		// removeFrom(config,d)
		// add d to configd
		// saveConfigd
		fmt.Printf("Setup device %s", ipAddress)
		saveConfig()
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

// Using a function copy a slice removing those unwanted
func filter(inp []device.DeviceInfo, test func(device.DeviceInfo) bool) (ret []device.DeviceInfo) {
	for _, d := range inp {
		if test(d) {
			ret = append(ret, d)
		}
	}
	return
}

func setupDevice(forThisIp string) (d device.DeviceInfo) {
	d.Ip = forThisIp
	d.Type = "samsungtv"
	return
}
