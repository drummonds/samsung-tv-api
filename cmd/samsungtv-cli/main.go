package main

import (
	"context"
	"time"
	"github.com/grandcat/zeroconf"
	"fmt"
	"log"
	"encoding/json"
	"encoding/base64"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
    samsung_tv_api "github.com/avbdr/samsung-tv-api/pkg/samsung-tv-api"
    "github.com/davecgh/go-spew/spew"
)

type Device struct {
	Name string `json:"name"`
	Mac string `json:"mac"`
	Ip string `json:"ip"`
	Token string `json:"token,omitempty"`
	Api *samsung_tv_api.SamsungTvClient `json:"-"`
}
var devices_ []Device


func toMap(data []string) (map[string]string) {
	result := make(map[string]string)
	for _, item := range data {
		item = strings.Trim(item, "\"")
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = parts[1]
	}
	return result
}

func zeroconfDisco() {
	serviceType := "_airplay._tcp"
	domain := "local."
	timeout := 1 * time.Second

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		fmt.Println("Failed to initialize resolver:", err.Error())
		return
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(d <-chan *zeroconf.ServiceEntry) {

		for device := range d {
			deviceMap := toMap(device.Text)
			if deviceMap["manufacturer"] != "Samsung" {
				continue
			}
			spew.Dump(device)
			tv := Device{
				Name: device.Instance,
				Mac: deviceMap["deviceid"],
				Ip: device.AddrIPv4[0].String(),
				Token: "",
			}
			devices_ = append(devices_, tv)
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = resolver.Browse(ctx, serviceType, domain, entries)
	if err != nil {
		fmt.Println("Failed to browse:", err.Error())
		return
	}
	<-ctx.Done()
}

func saveConfig() {
	if devices_ == nil {
		return
	}
	configBytes, err := json.MarshalIndent(devices_, "", "  ")
	if err != nil {
		fmt.Printf("1 %v", err)
	}
	fmt.Println(string(configBytes))
    homeDir, _ := os.UserHomeDir()
	err = ioutil.WriteFile(homeDir +"/.samsung.json", configBytes, 0644)
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


func main () {
	var tv Device
	deviceId := 0 
	if len(os.Args) == 1 {
		return
    }

	loadConfig()
	/*
	if len(os.Args) == 3 {
		deviceId = 0 // fixme. find the right tv from the config by name
		fmt.Printf("TV: %s", os.Args[2])
	}
	*/

	if os.Args[1] == "discover" {
		zeroconfDisco()
		saveConfig()
		return
	}

	tv = devices_[deviceId]

	tv.Api = samsung_tv_api.NewSamsungTvWebSocket(tv.Ip, tv.Token, 8002, 2, "RoomAI Remote", false)
	tv.Api.Mac = tv.Mac
	tv.Api.PowerOn()
	tv.Api.ConnectionSetup()
	tv.Api.Websocket.WaitFor("ms.channel.connect")

	if tv.Token == "" {
		devices_[deviceId].Token = tv.Api.GetToken()
		deviceInfo, deviceInfoErr := tv.Api.Rest.GetDeviceInfo()
		if deviceInfoErr == nil && deviceInfo.Device.NetworkType == "wireless" {
			devices_[deviceId].Mac = deviceInfo.Device.WifiMac
		}
		saveConfig()
	}

	if tv.Api == nil {
		log.Printf("tv is off")
		return
    }

	if os.Args[1] == "poweroff" {
		tv.Api.PowerOff()
		return
	}
	
	if os.Args[1] == "list" {
		apps, _ := tv.Api.Websocket.GetApplicationsList()
		for _, app := range apps.Data.Applications {
			log.Printf("%s - %s", app.AppID, app.Name)
		}
		return
	}

	if os.Args[1] == "open" {
		if len(os.Args) != 3 {
			log.Fatal("no key specified")
		}
		if strings.HasPrefix(os.Args[2], "http") {
			tv.Api.Websocket.OpenBrowser(os.Args[2])
		} else {
			tv.Api.Rest.RunApplication(os.Args[2])
		}
		return
	}
	if os.Args[1] == "key" {
		if len(os.Args) != 3 {
			log.Fatal("no key specified")
		}
		tv.Api.Websocket.SendClick(os.Args[2])
		return
	}
	if os.Args[1] == "volup" {
		tv.Api.Websocket.SendClick("KEY_VOLUP")
		return
	}

	if os.Args[1] == "voldown" {
		tv.Api.Websocket.SendClick("KEY_VOLDOWN")
		return
	}

	if os.Args[1] == "vol" {
		if len(os.Args) != 3 {
			vol, _ := tv.Api.Upnp.GetCurrentVolume()
			log.Printf("volume is %d", vol)
			return
		}
		intValue,_ := strconv.Atoi(os.Args[2])
		tv.Api.Upnp.SetVolume(intValue)
		return
	}

	if os.Args[1] == "test" {
		tv.Api.Websocket.WaitFor("qwesdfsf")
	}

	if os.Args[1] == "text" {
		if len(os.Args) != 3 {
			log.Fatal("no key specified")
		}
		tv.Api.Websocket.SendText(base64.StdEncoding.EncodeToString([]byte(os.Args[2])))
		return
	}

	if os.Args[1] == "stream" {
		if len(os.Args) != 3 {
			log.Fatal("no key specified")
		}
		log.Printf("Streaming %s", os.Args[2])
		tv.Api.Upnp.SetCurrentMedia(os.Args[2])
		return
	}

	if os.Args[1] == "info" {
		deviceInfo, deviceInfoErr := tv.Api.Rest.GetDeviceInfo()
		if deviceInfoErr != nil {
			fmt.Printf("error %#v", deviceInfoErr)
		}
		spew.Dump("%v", deviceInfo)
		log.Printf("OS %s", deviceInfo.Device.Os)
		return
	}
}
