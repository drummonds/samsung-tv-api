package upnp

import (
	"crypto/tls"
	"encoding/json"
    "encoding/xml"
	"fmt"
	xj "github.com/basgys/goxml2json"
	"log"
	"net"
	"time"
	"net/http"
	"net/url"
	"strconv"
	"strings"
    "io/ioutil"

)

// This package covers the support for the Universal Plug & Play (UPNP)

type UpnpClient struct {
	BaseUrl func(string) *url.URL
}

// makeSoapRequest will send a API http call (soap) to the given endpoint (base url + protocol).
// always being a POST. response will be converted to JSON and will be unmarshalled to the
// output interface.
//
// TODO
// 	* support binding to a non 200 response or determine the error message returned and use it in the error response
func (s *UpnpClient) makeSoapRequest(action, arguments, protocol string, output interface{}) error {
	u := s.BaseUrl(protocol + "1").String()
	fmt.Println(u)

	body := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"+
		"<s:Envelope xmlns:s=\"http://schemas.xmlsoap.org/soap/envelope/\" s:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\">\n"+
		"<s:Body>\n"+
		"<u:%s xmlns:u=\"urn:schemas-upnp-org:service:%s:1\">\n"+
		"<InstanceID>0</InstanceID>\n"+
		"%s\n"+
		"</u:%s>\n"+
		"</s:Body>\n"+
		"</s:Envelope>", action, protocol, arguments, action)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("POST", u, strings.NewReader(body))
	req.Header.Set("SOAPAction", fmt.Sprintf("\"urn:schemas-upnp-org:service:%s:1#%s\"", protocol, action))

	if err != nil {
		return err
	}

	resp, clientErr := client.Do(req)

	if clientErr != nil {
		return clientErr
	}

	defer resp.Body.Close()

	content, convertErr := xj.Convert(resp.Body)

	if convertErr != nil {
		return convertErr
	}

	return json.Unmarshal(content.Bytes(), &output)
}

// GetCurrentVolume returns the value of the current volume level
//
// TODO
// 	* This has to been tested with any bad input, should be regarded as not stable.
func (s *UpnpClient) GetCurrentVolume() (int, error) {
	log.Println("Get device volume via saop api")

	output := GetDeviceVolumeResponse{}
	err := s.makeSoapRequest("GetVolume", "<Channel>Master</Channel>", "RenderingControl", &output)

	if err != nil {
		return -1, err
	}

	return strconv.Atoi(output.Envelope.Body.GetVolumeResponse.CurrentVolume)
}

// SetVolume will update the current volume of the display to the provided value.
//
// TODO
// 	* This has to been tested with any bad input, should be regarded as not stable.
func (s *UpnpClient) SetVolume(volume int) error {
	log.Printf("set the volume of the tv to %d via soap api\n", volume)

	var output interface{}

	args := fmt.Sprintf("<Channel>Master</Channel><DesiredVolume>%d</DesiredVolume>", volume)
	return s.makeSoapRequest("SetVolume", args, "RenderingControl", &output)
}

// GetCurrentMuteStatus returns true if and only if the TV is currently muted
//
// TODO
// 	* This has to been tested with any bad input, should be regarded as not stable.
func (s *UpnpClient) GetCurrentMuteStatus() (bool, error) {
	log.Println("Get device mute status via saop api")

	output := GetCurrentMuteStatusResponse{}
	err := s.makeSoapRequest("GetMute", "<Channel>Master</Channel>", "RenderingControl", &output)

	if err != nil {
		return false, err
	}

	return output.Envelope.Body.GetMuteResponse.CurrentMute == "1", err
}

// SetCurrentMedia will tell the display to play the current media via the URL.
//
// TODO
// 	* This has to been tested with any bad input, should be regarded as not stable.
// 	* This requires to be tested, it has not been ran to close any applications yet.
func (s *UpnpClient) SetCurrentMedia(url string) error {
	args := fmt.Sprintf("<CurrentURI>%s</CurrentURI><CurrentURIMetaData></CurrentURIMetaData>", url)

	var output interface{}
	var err error

	err = s.makeSoapRequest("SetAVTransportURI", args, "AVTransport", &output)

	if err != nil {
		return err
	}

	return s.PlayCurrentMedia()
}

// GetCurrentMedia will return the status of the current media playing
//
// TODO
//  * This has to been tested with any bad input, should be regarded as not stable.
//  * This requires to be tested, it has not been ran to close any applications yet.
func (s *UpnpClient) GetCurrentMedia() (interface{}, error) {
    var output interface{}
    var err error

    err = s.makeSoapRequest("GetTransportInfo", "", "AVTransport", &output)

    if err != nil {
        return err, nil
    }
    return output, nil
}

// GetPositionInfo will return the status of the current media playing
func (s *UpnpClient) GetPositionInfo () (map[string]string, error) {
    var output GetPositionInfoResponse
    var err error

    err = s.makeSoapRequest("GetPositionInfo", "", "AVTransport", &output)

    if err != nil {
        return nil, err
    }

	didlXml := strings.Replace(output.Envelope.Body.GetPositionResponse.TrackMetaData, "&quot;", "\"", -1)
    didlXml = strings.Replace(didlXml, "&gt;", ">", -1)
	didlXml = strings.Replace(didlXml, "&lt;", "<", -1)
	didl :=  new(TrackMetaData_XML)
	err = xml.Unmarshal([]byte(didlXml), &didl)

	ret := map[string]string{
		"Track": output.Envelope.Body.GetPositionResponse.Track,
		"RelTime": output.Envelope.Body.GetPositionResponse.RelTime,
		"TrackDuration": output.Envelope.Body.GetPositionResponse.TrackDuration,
		"Artist": didl.Item.Creator,
		"Album": didl.Item.Album,
		"Cover": didl.Item.AlbumArtUri,
		"Uri": output.Envelope.Body.GetPositionResponse.TrackURI,
	}
	return ret, nil
}


// PlayCurrentMedia will attempt to play the current media already set on the display.
//
// TODO
// 	* This has to been tested with any bad input, should be regarded as not stable.
// 	* This requires to be tested, it has not been ran to close any applications yet.
func (s *UpnpClient) PlayCurrentMedia() error {
	var output interface{}
	return s.makeSoapRequest("Play", "<Speed>1</Speed>", "AVTransport", &output)
}

// Pause will attempt to pause playback.
//
func (s *UpnpClient) Pause () error {
	var output interface{}
	err := s.makeSoapRequest("Pause", "<Speed>1</Speed>", "AVTransport", &output)
	return err
}

// PlayNext will attempt to play the next media in playlist.
//
func (s *UpnpClient) PlayNext() error {
	var output interface{}
	return s.makeSoapRequest("Next", "", "AVTransport", &output)
}

// PlayPrev will attempt to play the next media in playlist.
//
func (s *UpnpClient) PlayPrevious() error {
	var output interface{}
	return s.makeSoapRequest("Previous", "", "AVTransport", &output)
}

func toMap(data []byte) (map[string]string) {
	headers := strings.Split(string(data), "\n")
	result := make(map[string]string)
	for _, item := range headers {
		item = strings.Trim(item, "\"")
		parts := strings.SplitN(item, ": ", 2)
		if len(parts) < 2 {
			continue
		}
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result
}

func Discover(filter string, manufacturer string, devType string) ([]map[string]string) {
    found := make([]map[string]string, 0)
	ssdpAddress := "239.255.255.250:1900"

	udpAddr, _ := net.ResolveUDPAddr("udp4", ssdpAddress)
	conn, err := net.ListenMulticastUDP("udp4", nil, udpAddr)
	if err != nil {
		fmt.Println("Error opening UDP connection:", err)
		return found
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	discoverMessage := []byte(
		"M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 1\r\n" +
		"ST: ssdp:all\r\n" +
		"\r\n",
	)
	_, err = conn.WriteToUDP(discoverMessage, udpAddr)
	if err != nil {
		fmt.Println("Error sending discover message:", err)
		return found
	}

	for {
		buffer := make([]byte, 2048)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				return found
			}
			fmt.Println("Error reading from UDP:", err)
			return found
		}
		data := toMap(buffer[:n])
		if data["ST"] != filter {
			continue
		}
		if data["LOCATION"]== "" {
			continue
		}
        parsedURL, _ := url.Parse(data["LOCATION"])
        props,err := DeviceProperties(data["LOCATION"])
        if err != nil {
			log.Printf("%v", err)
            continue
        }
		if props.Manufacturer != manufacturer {
			continue
		}
        d := make(map[string]string)
		if props.RoomName != "" {
			d["name"] = props.RoomName
		} else {
			d["name"] = props.FriendlyName
		}
        d["ip"] = parsedURL.Hostname()
        d["type"] = devType 
        found = append(found, d)
	}
	return found
}


func DeviceProperties(url string) (upnpDevice_XML, error) {
    resp, err := http.Get(url)
    if err != nil {
        return upnpDevice_XML{}, err
    }
    defer resp.Body.Close()
    
    var result upnpDescribeDevice_XML
    if body, err := ioutil.ReadAll(resp.Body); nil == err {
        xml.Unmarshal(body, &result)
        return result.Device[0], nil
    }
    return upnpDevice_XML{}, err
}
