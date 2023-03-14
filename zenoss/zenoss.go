package zenoss

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	// SeverityCritical defines critical severity level
	SeverityCritical = Severity("Critical")

	// SeverityError defines error severity level
	SeverityError = Severity("Error")

	// SeverityWarning defines warning severity level
	SeverityWarning = Severity("Warning")

	// SeverityInfo defines info severity level
	SeverityInfo = Severity("Info")

	// SeverityDebug defines debug severity level
	SeverityDebug = Severity("Debug")

	// SeverityClear defines clear severity level
	SeverityClear = Severity("Clear")
)

// Severity defines event severity
type Severity string

// Client allowing for perfoming operations against the Zenoss API
type Client interface {
	// AddEvent to component on device in Zenoss
	AddEvent(ctx context.Context, summary, message, device, component string, severity Severity, evClass, evKey string, extraData map[string]string) error
}

type client struct {
	client   *http.Client
	baseURI  string
	tid      int
	mTid     sync.Mutex
	username string
	password string
	monitor  string
}

type request struct {
	Action string `json:"action"`
	Method string `json:"method"`
	Data   []data `json:"data"`
	Tid    int    `json:"tid"`
}

type data map[string]string

type response struct {
	UUID   string                 `json:"uuid"`
	Action string                 `json:"action"`
	Method string                 `json:"method"`
	Result map[string]interface{} `json:"result"`
	Tid    int                    `json:"tid"`
}

// NewClient create new Zenoss instance
func NewClient(baseURI, username, password, monitor string, insecureSkipTLSVerify bool) (Client, error) {
	return &client{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipTLSVerify},
			},
		},
		baseURI:  baseURI,
		username: username,
		password: password,
		monitor:  monitor,
		tid:      0,
	}, nil
}

// AddEvent to component on device in Zenoss
func (api *client) AddEvent(ctx context.Context, summary, message, device, component string, severity Severity, evClass, evKey string, extraData map[string]string) error {
	logger := log.WithContext(ctx)
	d := map[string]string{
		"summary":    summary,
		"message":    message,
		"device":     device,
		"component":  component,
		"monitor":    api.monitor,
		"severity":   string(severity),
		"eventKey":   evKey,
		"evclasskey": "",
		"evclass":    evClass,
	}
	for k, v := range extraData {
		d[k] = v
	}
	r := request{
		Action: "EventsRouter",
		Method: "add_event",
		Data:   []data{d},
		Tid:    api.nextTid(),
	}
	logger.WithField("request", r).Trace("Constructed request for Zenoss")

	v, err := json.Marshal(r)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/evconsole_router", api.baseURI)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(v))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if len(api.username) > 0 || len(api.password) > 0 {
		req.SetBasicAuth(api.username, api.password)
	}
	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("unable to parse response from Zenoss: %w", err)
	}

	if !result.Result["success"].(bool) {
		return fmt.Errorf("event could not be created: %+v", result)
	}

	return nil
}

func (api *client) nextTid() int {
	api.mTid.Lock()
	defer api.mTid.Unlock()
	api.tid = api.tid + 1
	return api.tid
}
