package websocket

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/avbdr/samsung-tv-api/pkg/samsung-tv-api/keys"
	"golang.org/x/net/websocket"
	"log"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SamsungWebsocket struct {
	BaseUrl         func(string) *url.URL
	KeyPressDelay   int
	conn            *websocket.Conn
	readMutex       sync.Mutex
	writeMutex      sync.Mutex
	isReadWriteMode int32
}

type Request struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// OpenConnection will attempt to open a websocket connection with the TV.
// Reading the connecting JSON response from the TV.
//
// This will disable TLS validation on the self-signed certificate created
// and managed by the TV.
func (s *SamsungWebsocket) OpenConnection() (*ConnectionResponse, error) {
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}

	s.readMutex = sync.Mutex{}
	s.writeMutex = sync.Mutex{}

	origin := "http://localhost/"
	u := s.BaseUrl("samsung.remote.control").String()

	config, _ := websocket.NewConfig(u, origin)
	config.TlsConfig = &tls.Config{InsecureSkipVerify: true}

	ws, err := websocket.DialConfig(config)

	if err != nil {
		return nil, err
	}

	s.conn = ws

	var val ConnectionResponse
	readErr := s.readJSON(&val)

	return &val, readErr
}

func (s *SamsungWebsocket) WaitFor(event string) {
	fmt.Println("Waiting for ", event)
	origin := "http://localhost/"
	u := s.BaseUrl("samsung.remote.control").String()

	config, _ := websocket.NewConfig(u, origin)
	config.TlsConfig = &tls.Config{InsecureSkipVerify: true}

	var ws *websocket.Conn
	var err error

	for {
		ws, err = websocket.DialConfig(config)
		if err == nil {
			break
		}
	}
	defer ws.Close()

	for {
		var message string
		if err := websocket.Message.Receive(ws, &message); err != nil {
			log.Println("Error reading message:", err)
			break
		}
		// Log the received message
		log.Println("Received message:", message)
		if strings.Contains(message, event) {
			return
		}
	}
}

// sendJSONReceiveJSON will attempt to first send the command to the websocket server (TV)
// and then receive content back from the TV, unmarshalling the response to the output
// interface.
func (s *SamsungWebsocket) sendJSONReceiveJSON(command interface{}, output interface{}) error {
	s.writeMutex.Lock()
	s.readMutex.Lock()

	atomic.AddInt32(&s.isReadWriteMode, 1)

	defer func() {
		atomic.AddInt32(&s.isReadWriteMode, -1)
		s.writeMutex.Unlock()
		s.readMutex.Unlock()
	}()

	err := s.sendJSON(command)

	if err != nil {
		return err
	}
	return s.readJSON(output)
}

// sendJSON will convert the provided command interface to JSON and then
// into a byte array stream, sending it to the server.
func (s *SamsungWebsocket) sendJSON(command interface{}) error {
	if atomic.LoadInt32(&s.isReadWriteMode) != 1 {
		s.writeMutex.Lock()
		defer s.writeMutex.Unlock()
	}

	msg, err := json.Marshal(command)

	if err != nil {
		return err
	}

	_, err = s.conn.Write(msg)

	return err
}

// read will read the next frame of data from the websocket.
// Returning the byte array back.
func (s *SamsungWebsocket) read() ([]byte, error) {
	if atomic.LoadInt32(&s.isReadWriteMode) != 1 {
		fmt.Println("atomic read lock not set, locking read", atomic.LoadInt32(&s.isReadWriteMode))
		s.readMutex.Lock()
		defer s.readMutex.Unlock()
	}

	var data []byte
	err := websocket.Message.Receive(s.conn, &data)

	return data, err
}

// readJSON will read the next frame of data from the websocket and attempt
// to convert the content to JSON and unmarshal to the given type.
func (s *SamsungWebsocket) readJSON(val interface{}) error {
	msg, err := s.read()

	if err != nil {
		return err
	}

	return json.Unmarshal(msg, val)
}

// GetApplicationsList will request the TV to send over the listed applications
// currently on the TV.
//
// *Important*
// On newer models of TV, the process of which pulls the applications list is
// not installed, this can be installed for the device via the OS studio,
// which will require turning developer mode on within the TV.
//
// DOC: TODO
func (s *SamsungWebsocket) GetApplicationsList() (ApplicationsResponse, error) {
	log.Println("Getting applications lists via ws api")

	var output ApplicationsResponse

	var req = Request{
		Method: "ms.channel.emit",
		Params: map[string]interface{}{
			"event": "ed.installedApp.get",
			"to":    "host",
		},
	}

	for {
		err := s.sendJSONReceiveJSON(req, &output)
		if err != nil {
			return output, err
		}
		if output.Event == "ed.installedApp.get" {
			return output, err
		}
	}
}

// RunApplication will tell the TV via the web socket api to run a given application
// by using the provided application id.
//
// TODO
//   - This requires to be tested, it has not been ran to close any applications yet.
//   - This has to been tested with any bad input, should be regarded as not stable.
func (s *SamsungWebsocket) RunApplication(appId, appType, metaTag string) error {
	log.Printf("Running application %s via ws api\n", appId)

	if appType == "" {
		appType = "DEEP_LINK"
	}

	var req = Request{
		Method: "ms.channel.emit",
		Params: map[string]interface{}{
			"event": "ed.apps.launch",
			"to":    "host",
			"data": map[string]interface{}{
				// action_type: NATIVE_LAUNCH / DEEP_LINK
				// # app_type == 2 ? "DEEP_LINK" : "NATIVE_LAUNCH",
				"action_type": appType,
				"appId":       appId,
				"metaTag":     metaTag,
			},
		},
	}

	return s.sendJSON(req)
}

// SendClick will command the TV to perform a click on a given key.
//
// TODO
//   - This has to been tested with any bad input, should be regarded as not stable.
func (s *SamsungWebsocket) SendClick(key string) error {
	return s.SendKey(key, 1, "Click")
}

// SendKey will command the TV to perform a given action on a given key, this
// could include bing a click.
//
// TODO
//   - This has to been tested with any bad input, should be regarded as not stable.
func (s *SamsungWebsocket) SendKey(key string, times int, cmd string) error {

	if cmd == "" {
		cmd = "Click"
	}

	log.Printf("Sending key %s with command %s, %d times via ws api\n", key, cmd, times)

	for i := 0; i < times; i++ {
		log.Printf("Sending key %s via ws api\n", key)

		var req = Request{
			Method: "ms.remote.control",
			Params: map[string]interface{}{
				"Cmd":          cmd,
				"DataOfCmd":    key,
				"Option":       "false",
				"TypeOfRemote": "SendRemoteKey",
			},
		}

		err := s.sendJSON(req)

		if err != nil {
			return err
		}

		time.Sleep(time.Duration(s.KeyPressDelay) * time.Millisecond)
	}

	return nil
}

func (s *SamsungWebsocket) SendText(text string) error {
	log.Printf("Sending text %s via ws api\n", text)
	var req = Request{
		Method: "ms.remote.control",
		Params: map[string]interface{}{
			"Cmd":          text,
			"DataOfCmd":    "base64",
			"TypeOfRemote": "SendInputString",
		},
	}

	err := s.sendJSON(req)

	if err != nil {
		return err
	}

	return nil
}

// HoldKey will command the TV to press a given key and then wait the provided
// seconds until commanding the TV to release that given key again.
//
// TODO
//   - This requires to be tested, it has not been ran to close any applications yet.
//   - This has to been tested with any bad input, should be regarded as not stable.
func (s *SamsungWebsocket) HoldKey(key string, seconds int) error {
	log.Printf("Sending hold key %s for %d seconds via ws api\n", key, seconds)

	pressErr := s.SendKey(key, 1, "Press")

	if pressErr != nil {
		return pressErr
	}

	time.Sleep(time.Duration(seconds) * time.Second)

	log.Printf("Sending release key %s via ws api\n", key)
	releaseErr := s.SendKey(key, 1, "Release")

	if releaseErr != nil {
		return releaseErr
	}

	return nil
}

// ChangeChannel will convert the provided channel numbers into key presses and
// send these key presses to the TV. Ensuring to send enter after completion.
//
// TODO
//   - This has to been tested with any bad input, should be regarded as not stable.
func (s *SamsungWebsocket) ChangeChannel(channel string) error {
	split := strings.Split(channel, "")

	for _, digit := range split {
		err := s.SendKey(fmt.Sprintf("KEY_%s", digit), 1, "Click")
		if err != nil {
			return err
		}
	}

	return s.SendKey(keys.NavigationEnter, 1, "Click")
}

// MoveCursor will command the TV to move the mouse cursor to a given X,Y position
// over the provided duration.
//
// TODO
//   - This requires to be tested, it has not been ran to close any applications yet.
//   - This has to been tested with any bad input, should be regarded as not stable.
func (s *SamsungWebsocket) MoveCursor(x, y, duration int) error {
	log.Printf("Sending move Cursor to x: %d, y: %d for duration %d via ws api\n", x, y, duration)

	var req = Request{
		Method: "ms.remote.control",
		Params: map[string]interface{}{
			"Cmd":          "Move",
			"TypeOfRemote": "ProcessMouseDevice",
			"Position": map[string]interface{}{
				"x":    x,
				"y":    y,
				"Time": string(rune(duration)),
			},
		},
	}

	return s.sendJSON(req)
}

// OpenBrowser will command the TV to open a given URL within the browser.
//
// TODO
//   - This requires to be tested, it has not been ran to close any applications yet.
//   - This has to been tested with any bad input, should be regarded as not stable.
func (s *SamsungWebsocket) OpenBrowser(url string) error {
	log.Printf("opening browser to url: %s via ws api\n", url)
	return s.RunApplication("org.tizen.browser", "NATIVE_LAUNCH", url)
}

func (s *SamsungWebsocket) Disconnect() error {
	return s.conn.Close()
}
