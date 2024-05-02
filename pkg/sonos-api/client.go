package sonos

import (
    "github.com/avbdr/samsung-tv-api/pkg/upnp"
	"github.com/ianr0bkny/go-sonos/ssdp"
	"log"
	"fmt"
	"net/url"
)

type SonosClient struct {
    host string
    Upnp upnp.UpnpClient
}


func NewSonosDevice(host string) *SonosClient {
    client := &SonosClient{host: host}
    client.Upnp = upnp.UpnpClient{
        BaseUrl: func(endpoint string) *url.URL {
            return &url.URL{     
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", host, 1400),
				Path:   fmt.Sprintf("MediaRenderer/AVTransport/Control"),
			}
        },
    }
	return client
}

func Discover() ([]map[string]string) {
	found := make([]map[string]string, 0)
	mgr := ssdp.MakeManager()

	// Discover()
	//  eth0 := Network device to query for UPnP devices
	// 11209 := Free local port for discovery replies
	// false := Do not subscribe for asynchronous updates
	mgr.Discover("wlp2s0", "11209", false)

	// SericeQueryTerms
	// A map of service keys to minimum required version
	qry := ssdp.ServiceQueryTerms{
		ssdp.ServiceKey("schemas-upnp-org-MusicServices"): -1,
	}

	// Look for the service keys in qry in the database of discovered devices
	result := mgr.QueryServices(qry)
	if dev_list, has := result["schemas-upnp-org-MusicServices"]; has {
		for _, dev := range dev_list {
            if dev.Product() != "Sonos" {
                continue
            }
            parsedURL, _ := url.Parse(string(dev.Location()))
            props,err := upnp.DeviceProperties(string(dev.Location()))
            if err != nil {
                log.Printf("%v", err)
                continue
            }          
            d := make(map[string]string)
            d["name"] = props.RoomName
            d["ip"] = parsedURL.Hostname()
            d["type"] = "sonos"
            found = append(found, d)
		}
	}
	mgr.Close()
	return found
}

func (c *SonosClient) GetToken() string {
	return ""
}

func (c *SonosClient) Init() {
}

func (c *SonosClient) PowerOff() error {
	return nil
}

func (c *SonosClient) PowerOn() error {
	return nil
}

func (c *SonosClient) List() error {
	return nil
}

func (c *SonosClient) Open(url string) error {
	return nil
}

func (c *SonosClient) Key(key string) error {
	return nil
}

func (c *SonosClient) VolUp() error {
	return nil
}

func (c *SonosClient) VolDown() error  {
	return nil
}

func (c *SonosClient) Vol(vol int) (int, error) {
	client := upnp.UpnpClient{
			BaseUrl: func(endpoint string) *url.URL {
				return &url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("%s:%d", c.host, 1400),
					Path:   "MediaRenderer/RenderingControl/Control",
				}
			},
	}

    if (vol == -1) {
        vol, err := client.GetCurrentVolume()
        return vol, err
    }
    client.SetVolume(vol)
    return vol, nil
}

func (c *SonosClient) Test() error {
	return nil
}

func (c *SonosClient) Text(text string) error  {
	return nil
}

func (c *SonosClient) Stream(url string) error  {
	return nil
}

func (c *SonosClient) Info() (string, error) {
	c.Upnp.GetCurrentMedia()
	return "", nil
}

func (c *SonosClient) Next() error  {
    c.Upnp.PlayNext()
	return nil
}

func (c *SonosClient) Prev() error  {
    c.Upnp.PlayPrevious()
	return nil
}

func (c *SonosClient) Pause() error {
    c.Upnp.Pause()
	return nil
}

func (c *SonosClient) Play() error  {
    c.Upnp.PlayCurrentMedia()
	return nil
}

func (c *SonosClient) Status() (interface{}, error)  {
    out, err := c.Upnp.GetCurrentMedia()
	log.Printf("%#v", out)
    out, err = c.Upnp.GetPositionInfo()
	log.Printf("%#v", out)
    return out, err
}

