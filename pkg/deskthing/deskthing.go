package deskthing

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DeskthingListener defines the function signature for event listeners.
type DeskthingListener func(args ...interface{})

// IncomingEvent represents events coming from the server.
type IncomingEvent string

const (
	IncomingEventMessage      IncomingEvent = "message"
	IncomingEventData         IncomingEvent = "data"
	IncomingEventGet          IncomingEvent = "get"
	IncomingEventSet          IncomingEvent = "set"
	IncomingEventCallbackData IncomingEvent = "callback-data"
	IncomingEventStart        IncomingEvent = "start"
	IncomingEventStop         IncomingEvent = "stop"
	IncomingEventPurge        IncomingEvent = "purge"
	IncomingEventInput        IncomingEvent = "input"
	IncomingEventAction       IncomingEvent = "action"
	IncomingEventConfig       IncomingEvent = "config"
	IncomingEventSettings     IncomingEvent = "settings"
)

// OutgoingEvent represents events that can be sent back to the server.
type OutgoingEvent string

const (
	OutgoingEventMessage OutgoingEvent = "message"
	OutgoingEventData    OutgoingEvent = "data"
	OutgoingEventGet     OutgoingEvent = "get"
	OutgoingEventSet     OutgoingEvent = "set"
	OutgoingEventAdd     OutgoingEvent = "add"
	OutgoingEventOpen    OutgoingEvent = "open"
	OutgoingEventToApp   OutgoingEvent = "toApp"
	OutgoingEventError   OutgoingEvent = "error"
	OutgoingEventLog     OutgoingEvent = "log"
	OutgoingEventAction  OutgoingEvent = "action"
	OutgoingEventButton  OutgoingEvent = "button"
)

// SongData represents song-related data.
type SongData struct {
	Album           *string  `json:"album"`
	Artist          *string  `json:"artist"`
	Playlist        *string  `json:"playlist"`
	PlaylistID      *string  `json:"playlist_id"`
	TrackName       string   `json:"track_name"`
	ShuffleState    *bool    `json:"shuffle_state"`
	RepeatState     string   `json:"repeat_state"` // "off", "all", "track"
	IsPlaying       bool     `json:"is_playing"`
	CanFastForward  bool     `json:"can_fast_forward"`
	CanSkip         bool     `json:"can_skip"`
	CanLike         bool     `json:"can_like"`
	CanChangeVolume bool     `json:"can_change_volume"`
	CanSetOutput    bool     `json:"can_set_output"`
	TrackDuration   *float64 `json:"track_duration"`
	TrackProgress   *float64 `json:"track_progress"`
	Volume          float64  `json:"volume"`
	Thumbnail       *string  `json:"thumbnail"`
	Device          *string  `json:"device"`
	ID              *string  `json:"id"`
	DeviceID        *string  `json:"device_id"`
}

// GetTypes represents sub-types for the 'get' event.
type GetTypes string

const (
	GetTypeData   GetTypes = "data"
	GetTypeConfig GetTypes = "config"
	GetTypeInput  GetTypes = "input"
)

// Manifest represents application metadata.
type Manifest struct {
	Type        []string `json:"type"`
	Requires    []string `json:"requires"`
	Label       string   `json:"label"`
	Version     string   `json:"version"`
	Description *string  `json:"description,omitempty"`
	Author      *string  `json:"author,omitempty"`
	ID          string   `json:"id"`
	IsWebApp    bool     `json:"isWebApp"`
	IsLocalApp  bool     `json:"isLocalApp"`
	Platforms   []string `json:"platforms"`
	Homepage    *string  `json:"homepage,omitempty"`
	Repository  *string  `json:"repository,omitempty"`
}

// AuthScope represents a single authentication scope.
type AuthScope struct {
	Instructions string  `json:"instructions"`
	Label        string  `json:"label"`
	Value        *string `json:"value,omitempty"`
}

// AuthScopes maps scope identifiers to their definitions.
type AuthScopes map[string]AuthScope

// SettingsNumber represents a numeric setting.
type SettingsNumber struct {
	Value       float64 `json:"value"`
	Type        string  `json:"type"` // "number"
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
}

// SettingsBoolean represents a boolean setting.
type SettingsBoolean struct {
	Value       bool    `json:"value"`
	Type        string  `json:"type"` // "boolean"
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
}

// SettingsString represents a string setting.
type SettingsString struct {
	Value       string  `json:"value"`
	Type        string  `json:"type"` // "string"
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
}

// Option represents an option in select settings.
type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// SettingsSelect represents a select setting.
type SettingsSelect struct {
	Value       string   `json:"value"`
	Type        string   `json:"type"` // "select"
	Label       string   `json:"label"`
	Description *string  `json:"description,omitempty"`
	Options     []Option `json:"options"`
}

// SettingsMultiSelect represents a multiselect setting.
type SettingsMultiSelect struct {
	Value       []string `json:"value"`
	Type        string   `json:"type"` // "multiselect"
	Label       string   `json:"label"`
	Description *string  `json:"description,omitempty"`
	Options     []Option `json:"options"`
}

// SettingsType is a union of different setting types.
type SettingsType interface{}

// AppSettings maps setting identifiers to their definitions.
type AppSettings map[string]SettingsType

// InputResponse represents user input responses.
type InputResponse map[string]interface{}

// SocketData represents data sent over the socket.
type SocketData struct {
	App     *string     `json:"app,omitempty"`
	Type    string      `json:"type"`
	Request *string     `json:"request,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// DataInterface represents the main data structure.
type DataInterface struct {
	Settings AppSettings            `json:"settings,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// OutgoingData represents data sent to the server.
type OutgoingData struct {
	Type    OutgoingEvent `json:"type"`
	Request string        `json:"request"`
	Payload interface{}   `json:"payload"`
}

// IncomingData represents data received from the server.
type IncomingData struct {
	Type    IncomingEvent `json:"type"`
	Request string        `json:"request"`
	Payload interface{}   `json:"payload"`
}

// ToServerFunc defines the function signature for sending data to the server.
type ToServerFunc func(data OutgoingData)

// SysEventsFunc defines the function signature for system events.
type SysEventsFunc func(event string, listener DeskthingListener) func()

// StartData represents the data required to start the application.
type StartData struct {
	ToServer  ToServerFunc
	SysEvents SysEventsFunc
}

// ValueTypes represents possible value types.
type ValueTypes interface{}

// EventFlavor represents different event flavors.
type EventFlavor int

const (
	EventFlavorKeyUp EventFlavor = iota
	EventFlavorKeyDown
	EventFlavorScrollUp
	EventFlavorScrollDown
	EventFlavorScrollLeft
	EventFlavorScrollRight
	EventFlavorSwipeUp
	EventFlavorSwipeDown
	EventFlavorSwipeLeft
	EventFlavorSwipeRight
	EventFlavorPressShort
	EventFlavorPressLong
)

// Action represents an action that can be registered.
type Action struct {
	Name              *string       `json:"name,omitempty"`
	Description       *string       `json:"description,omitempty"`
	ID                string        `json:"id"`
	Value             *ValueTypes   `json:"value,omitempty"`
	ValueOptions      *[]ValueTypes `json:"value_options,omitempty"`
	ValueInstructions *string       `json:"value_instructions,omitempty"`
	Icon              *string       `json:"icon,omitempty"`
	Source            string        `json:"source"`
	Version           string        `json:"version"`
	Enabled           bool          `json:"enabled"`
}

// Key represents a key that can be registered.
type Key struct {
	ID          string        `json:"id"`
	Source      string        `json:"source"`
	Description *string       `json:"description,omitempty"`
	Version     string        `json:"version"`
	Enabled     bool          `json:"enabled"`
	Flavors     []EventFlavor `json:"flavors"`
}

// ActionCallback represents a callback for an action.
type ActionCallback struct {
	ID    string     `json:"id"`
	Value ValueTypes `json:"value"`
}

// Response represents a generic response structure.
type Response struct {
	Data       interface{} `json:"data"`
	Status     int         `json:"status"`
	StatusText string      `json:"statusText"`
	Request    []string    `json:"request"`
}

// DeskThing is the main class managing events, data, and communication.
type DeskThing struct {
	listeners          map[IncomingEvent][]DeskthingListener
	manifest           *Manifest
	toServer           ToServerFunc
	sysEvents          SysEventsFunc
	sysListeners       []func()
	data               *DataInterface
	backgroundTasks    []func()
	isDataBeingFetched bool
	dataFetchQueue     []chan *DataInterface
	stopRequested      bool
	mutex              sync.Mutex
}

var (
	deskThingInstance *DeskThing
	once              sync.Once
)

// GetInstance returns the singleton instance of
func GetInstance() *DeskThing {
	once.Do(func() {
		deskThingInstance = &DeskThing{
			listeners:      make(map[IncomingEvent][]DeskthingListener),
			dataFetchQueue: []chan *DataInterface{},
		}
		deskThingInstance.loadManifest()
	})
	return deskThingInstance
}

// loadManifest loads the manifest file.
func (dt *DeskThing) loadManifest() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Failed to get executable path:", err)
		return
	}
	dir := filepath.Dir(exePath)
	manifestPath := filepath.Join(dir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Println("Failed to read manifest:", err)
		return
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		fmt.Println("Failed to parse manifest:", err)
		return
	}

	dt.manifest = &manifest
}

// On registers an event listener.
func (dt *DeskThing) On(event IncomingEvent, callback DeskthingListener) func() {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	dt.listeners[event] = append(dt.listeners[event], callback)

	return func() {
		dt.Off(event, callback)
	}
}

// Off removes an event listener.
func (dt *DeskThing) Off(event IncomingEvent, callback DeskthingListener) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	listeners := dt.listeners[event]
	for i, l := range listeners {
		if &l == &callback {
			dt.listeners[event] = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
}

// Once registers a one-time event listener.
func (dt *DeskThing) Once(event IncomingEvent, callback DeskthingListener) {
	var wrapper DeskthingListener
	wrapper = func(args ...interface{}) {
		dt.Off(event, wrapper)
		callback(args...)
	}
	dt.On(event, wrapper)
}

// notifyListeners notifies all listeners of a particular event.
func (dt *DeskThing) notifyListeners(event IncomingEvent, args ...interface{}) {
	dt.mutex.Lock()
	listeners := dt.listeners[event]
	dt.mutex.Unlock()

	for _, callback := range listeners {
		callback(args...)
	}
}

// SendData sends data to the server.
func (dt *DeskThing) SendData(event OutgoingEvent, payload interface{}, request string) {
	if dt.toServer == nil {
		fmt.Println("toServer function is not defined")
		return
	}

	outgoingData := OutgoingData{
		Type:    event,
		Request: request,
		Payload: payload,
	}

	dt.toServer(outgoingData)
}

// SendMessage sends a plain text message to the server.
func (dt *DeskThing) SendMessage(message string) {
	dt.SendData(OutgoingEventMessage, message, "")
}

// SendLog sends a log message to the server.
func (dt *DeskThing) SendLog(message string) {
	dt.SendData(OutgoingEventLog, message, "")
}

// SendError sends an error message to the server.
func (dt *DeskThing) SendError(message string) {
	dt.SendData(OutgoingEventError, message, "")
}

// OpenURL requests the server to open a specified URL.
func (dt *DeskThing) OpenURL(url string) {
	dt.SendData(OutgoingEventOpen, url, "")
}

// GetData fetches data from the server or returns cached data.
func (dt *DeskThing) GetData() (*DataInterface, error) {
	dt.mutex.Lock()
	if dt.data != nil {
		defer dt.mutex.Unlock()
		return dt.data, nil
	}

	if dt.isDataBeingFetched {
		ch := make(chan *DataInterface)
		dt.dataFetchQueue = append(dt.dataFetchQueue, ch)
		dt.mutex.Unlock()
		data := <-ch
		return data, nil
	}

	dt.isDataBeingFetched = true
	dt.mutex.Unlock()

	dt.SendData(OutgoingEventGet, nil, string(GetTypeData))

	timeout := time.After(5 * time.Second)
	ch := make(chan *DataInterface, 1)

	dt.Once(IncomingEventData, func(args ...interface{}) {
		if data, ok := args[0].(*DataInterface); ok {
			ch <- data
		} else {
			ch <- nil
		}
	})

	select {
	case data := <-ch:
		dt.mutex.Lock()
		dt.isDataBeingFetched = false
		dt.data = data
		for _, q := range dt.dataFetchQueue {
			q <- data
		}
		dt.dataFetchQueue = nil
		dt.mutex.Unlock()
		return data, nil
	case <-timeout:
		dt.mutex.Lock()
		dt.isDataBeingFetched = false
		for _, q := range dt.dataFetchQueue {
			q <- nil
		}
		dt.dataFetchQueue = nil
		dt.mutex.Unlock()
		return nil, errors.New("data fetch timed out")
	}
}

// AddSettings adds new settings or overwrites existing ones.
func (dt *DeskThing) AddSettings(settings AppSettings) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	if dt.data == nil {
		dt.data = &DataInterface{
			Settings: make(AppSettings),
		}
	}

	if dt.data.Settings == nil {
		dt.data.Settings = make(AppSettings)
	}

	for id, setting := range settings {
		dt.data.Settings[id] = setting
	}

	dt.SendData(OutgoingEventAdd, map[string]interface{}{"settings": dt.data.Settings}, "")
}

// RegisterAction registers a new action to the server.
func (dt *DeskThing) RegisterAction(name, id, description, flair string) {
	action := map[string]interface{}{
		"name":        name,
		"id":          id,
		"description": description,
		"flair":       flair,
	}
	dt.SendData(OutgoingEventAction, action, "add")
}

// Start initializes the DeskThing instance.
func (dt *DeskThing) Start(data StartData) error {
	dt.toServer = data.ToServer
	dt.sysEvents = data.SysEvents
	dt.stopRequested = false

	dt.notifyListeners(IncomingEventStart)
	return nil
}

// Stop stops the DeskThing instance.
func (dt *DeskThing) Stop() error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	dt.notifyListeners(IncomingEventStop)
	dt.stopRequested = true

	// Stop background tasks
	for _, cancel := range dt.backgroundTasks {
		cancel()
	}
	dt.backgroundTasks = nil

	return nil
}

// ToClient handles data received from the server.
func (dt *DeskThing) ToClient(data IncomingData) {
	if data.Type == IncomingEventData {
		if payload, ok := data.Payload.(map[string]interface{}); ok {
			dt.mutex.Lock()
			dt.data = &DataInterface{
				Data: payload,
			}
			dt.mutex.Unlock()
			dt.notifyListeners(IncomingEventData, dt.data)
		} else {
			dt.SendLog("Received invalid data from server")
		}
	} else {
		dt.notifyListeners(data.Type, data)
	}
}

// EncodeImageFromURL encodes an image from a URL and returns a base64 string.
func (dt *DeskThing) EncodeImageFromURL(url string, imageType string, retries int) (string, error) {
	if retries <= 0 {
		return "", errors.New("maximum retries reached")
	}

	resp, err := http.Get(url)
	if err != nil {
		return dt.EncodeImageFromURL(url, imageType, retries-1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return dt.EncodeImageFromURL(url, imageType, retries-1)
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	base64Str := fmt.Sprintf("data:image/%s;base64,%s", imageType, base64.StdEncoding.EncodeToString(imgData))
	return base64Str, nil
}
