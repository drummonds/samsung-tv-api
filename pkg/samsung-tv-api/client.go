package samsung_tv_api

import (
	"encoding/base64"
	"fmt"
	"github.com/avbdr/samsung-tv-api/internal/app/samsung-tv-api/wol"
	samsung_http "github.com/avbdr/samsung-tv-api/pkg/samsung-tv-api/http"
	"github.com/avbdr/samsung-tv-api/pkg/samsung-tv-api/websocket"
	"github.com/avbdr/samsung-tv-api/pkg/device"
	"github.com/avbdr/samsung-tv-api/pkg/upnp"
    "github.com/ianr0bkny/go-sonos/ssdp"
	"log"
	"strings"
	"time"
	"net/url"
)

type SamsungTvClient struct {
	Rest      samsung_http.SamsungRestClient
	Websocket websocket.SamsungWebsocket
	Upnp      upnp.UpnpClient
	port      int
	cfg       *device.DeviceInfo
	keyPressDelay int
	name      string
}

func NewSamsungTvWebSocket(cfg *device.DeviceInfo, keyPressDelay int, autoConnect bool) *SamsungTvClient {
	if keyPressDelay == 0 {
		keyPressDelay = 1
	}
	client := &SamsungTvClient{
		name:          "RoomsAI Remote",
		cfg:		   cfg,
		port:          8002,
		keyPressDelay: keyPressDelay,
	}

	client.Rest = samsung_http.SamsungRestClient{
		BaseUrl: func(endpoint string) *url.URL {
			return client.formatRestUrl(endpoint)
		},
	}

	client.Websocket = websocket.SamsungWebsocket{
		BaseUrl: func(endpoint string) *url.URL {
			return client.formatWebSocketUrl(endpoint)
		},
		KeyPressDelay: keyPressDelay,
	}

	client.Upnp = upnp.UpnpClient{
		BaseUrl: func(endpoint string) *url.URL {
			return client.formatUpnpUrl(endpoint)
		},
	}

	if autoConnect {
		if err := client.ConnectionSetup(); err != nil {
			log.Fatalln(err)
		}
	}

	return client
}

// ConnectionSetup will attempt to open a connection to the websocket endpoint on
// the TV while after connecting, update the internal token to the newest value
// regardless if its the same.
func (s *SamsungTvClient) ConnectionSetup() error {
	wsResp, err := s.Websocket.OpenConnection()

	if err != nil {
		return err
	}

	if len(wsResp.Data.Clients) > 0 && wsResp.Data.Token != "" {
		s.cfg.Token = wsResp.Data.Token
	}

	return nil
}

// isSslConnection returns true if and only if the port is the SSL port for the
// connection otherwise it is not configured for SSL.
func (s *SamsungTvClient) isSslConnection() bool {
	return s.port == 8002
}

// formatWebSocketUrl returns the formatted web socket url for connecting
func (s *SamsungTvClient) formatWebSocketUrl(endpoint string) *url.URL {
	if endpoint != "" && string(endpoint[0]) != "/" {
		endpoint = "/" + endpoint
	}

	name := base64.StdEncoding.EncodeToString([]byte(s.name))

	u := &url.URL{
		Scheme:   "ws",
		Host:     fmt.Sprintf("%s:%d", s.cfg.Ip, s.port),
		Path:     fmt.Sprintf("api/v2/channels%s", endpoint),
		RawQuery: fmt.Sprintf("name=%s", name),
	}

	if s.isSslConnection() {
		u.Scheme += "s"
		u.RawQuery += fmt.Sprintf("&token=%s", s.cfg.Token)
	}

	return u
}

// formatRestUrl returns the formatted rest api url for connecting to
// the tv rest service
func (s *SamsungTvClient) formatRestUrl(endpoint string) *url.URL {
	if endpoint != "" && string(endpoint[0]) != "/" {
		endpoint = "/" + endpoint
	}

	if endpoint == "" || string(endpoint[len(endpoint)-1]) != "/" {
		endpoint += "/"
	}

	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", s.cfg.Ip, s.port),
		Path:   fmt.Sprintf("api/v2%s", endpoint),
	}

	if s.isSslConnection() {
		u.Scheme += "s"
	}

	return u
}

// formatUpnpUrl returns the formatted api url for connecting to
// the tv soap service
func (s *SamsungTvClient) formatUpnpUrl(endpoint string) *url.URL {
	if endpoint != "" && string(endpoint[0]) != "/" {
		endpoint = "/" + endpoint
	}

	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", s.cfg.Ip, 9197),
		Path:   fmt.Sprintf("upnp/control%s", endpoint),
	}

	return u
}

func (s *SamsungTvClient) Disconnect() error {
	return s.Websocket.Disconnect()
}

// GetToken returns the current Auth token used by the client.
func (s *SamsungTvClient) GetToken() string {
	return s.cfg.Token
}

// WakeOnLan broadcasts a magic packet to all listening devices with the target
// mac address being the device (provided) thus telling the TV to turn on.
func WakeOnLan(mac string) error {
	packet, err := wol.NewMagicPacket(mac)

	if err != nil {
		return err
	}

	return packet.Send("255.255.255.255")
}

func (s *SamsungTvClient) IsAlive () (bool) {
    _, deviceInfoErr := s.Rest.GetDeviceInfo()
    if deviceInfoErr != nil {
        return false
    }
    return true
}

func (s *SamsungTvClient) PowerOn () error {
    if s.IsAlive() {
        s.ConnectionSetup()
		// turned off TV reports volume -1
        vol, _ := s.Upnp.GetCurrentVolume()
        if vol == -1 {
            s.Websocket.SendClick("KEY_POWER")
        }
    }
	log.Printf("wol to %s", s.cfg.Mac)
    WakeOnLan(s.cfg.Mac)
    for s.IsAlive() == false {
        time.Sleep(500 * time.Millisecond)
    }
	return nil
}

func (s *SamsungTvClient) PowerOff () error {
	s.Websocket.SendClick("KEY_POWER")
	return nil
}

func Discover() ([]map[string]string) {
	found := make([]map[string]string, 0)

    mgr := ssdp.MakeManager()
    defer mgr.Close()

    // Discover()
    //  eth0 := Network device to query for UPnP devices
    // 11209 := Free local port for discovery replies
    // false := Do not subscribe for asynchronous updates
    mgr.Discover("wlp2s0", "11209", false)

    // A map of service keys to minimum required version
    qry := ssdp.ServiceQueryTerms{
        ssdp.ServiceKey("dial-multiscreen-org-dial"): -1,
    }

    // Look for the service keys in qry in the database of discovered devices
    result := mgr.QueryServices(qry)
    if dev_list, has := result["dial-multiscreen-org-dial"]; has {
        for _, dev := range dev_list {
			if dev.Product() != "Samsung_UPnP_SDK" {
				continue
			}
			parsedURL, _ := url.Parse(string(dev.Location()))
			props,err := upnp.DeviceProperties(string(dev.Location()))
			if err != nil {
				log.Printf("%v", err)
				continue
			}
			d := make(map[string]string)
			d["name"] = props.FriendlyName
			d["ip"] = parsedURL.Hostname()
			d["type"] = "samsungtv"
			found = append(found, d)
        }
    }
	return found
}

func (s *SamsungTvClient) Init() {
    s.PowerOn()
    s.ConnectionSetup()
    s.Websocket.WaitFor("ms.channel.connect")
	if s.cfg.Mac == "" {
		deviceInfo, deviceInfoErr := s.Rest.GetDeviceInfo()
        if deviceInfoErr == nil && deviceInfo.Device.NetworkType == "wireless" {
			s.cfg.Mac = deviceInfo.Device.WifiMac
        }
    }
}

func (s *SamsungTvClient) List() error {
    apps, _ := s.Websocket.GetApplicationsList()
    for _, app := range apps.Data.Applications {
		log.Printf("%s - %s", app.AppID, app.Name)
    }
	return nil
}

func (s *SamsungTvClient) Open(url string) error {
    if strings.HasPrefix(url, "http") {
		s.Websocket.OpenBrowser(url)
    } else {
		s.Rest.RunApplication(url)
    }
	return nil
}

func (s *SamsungTvClient) Key(key string) error {
	s.Websocket.SendClick(key)
	return nil
}

func (s *SamsungTvClient) VolUp() error {
	s.Websocket.SendClick("KEY_VOLUP")
	return nil
}

func (s *SamsungTvClient) VolDown() error  {
	s.Websocket.SendClick("KEY_VOLDOWN")
	return nil
}

func (s *SamsungTvClient) Vol(vol int) (int, error) {
	if (vol == -1) {
		vol, err := s.Upnp.GetCurrentVolume()
		return vol, err
	}
	s.Upnp.SetVolume(vol)
	return vol, nil
}

func (s *SamsungTvClient) Test() error {
    s.Websocket.WaitFor("qwesdfsf")
	return nil
}

func (s *SamsungTvClient) Text(text string) error  {
    s.Websocket.SendText(base64.StdEncoding.EncodeToString([]byte(text)))
	return nil
}

func (s *SamsungTvClient) Stream(url string) error  {
	s.Upnp.SetCurrentMedia(url)
	return nil
}

func (s *SamsungTvClient) Info() (string, error) {
    return "", nil//s.Rest.GetDeviceInfo()
}

func (s *SamsungTvClient) Next() error  {
    s.Upnp.PlayNext()
	return nil
}

func (s *SamsungTvClient) Prev() error  {
    s.Upnp.PlayPrevious()
	return nil
}

func (s *SamsungTvClient) Pause() error {
	s.Upnp.Pause()
	return nil
}

func (s *SamsungTvClient) Play() error  {
	s.Upnp.PlayCurrentMedia()
	return nil
}

func (s *SamsungTvClient) Status() (interface{}, error)  {
    out, err := s.Upnp.GetCurrentMedia()
    log.Printf("%#v", out)
    out, err = s.Upnp.GetPositionInfo()
    log.Printf("%#v", out)
    return out, err
}
