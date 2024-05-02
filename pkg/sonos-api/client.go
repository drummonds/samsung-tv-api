package sonos

import (
    "github.com/avbdr/samsung-tv-api/pkg/upnp"
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
	return upnp.Discover("urn:schemas-upnp-org:service:MusicServices:1", "Sonos, Inc.", "sonos")
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

