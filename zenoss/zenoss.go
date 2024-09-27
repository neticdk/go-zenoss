package zenoss

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client allowing for perfoming operations against the Zenoss API
type Client interface {
	// AddEvent to component on device in Zenoss
	AddEvent(ctx context.Context, summary, message, device, component string, severity Severity, evClass, evKey string, extraData map[string]string) error

	// ReadDevice returns information on the given device uid
	ReadDevice(ctx context.Context, uid string) (*Device, error)

	// CreateDevice creates new in Zenoss
	CreateDevice(ctx context.Context, dev NewDevice) (*Device, error)

	// DeleteDevice deletes the device with the given uid
	DeleteDevice(ctx context.Context, uid string) error

	// UpdateDeviceProductionState updates only the production state of the given device uid
	UpdateDeviceProductionState(ctx context.Context, uid string, state int) error

	// ReadCustomProperty reads the names custom property of the given device
	ReadCustomProperty(ctx context.Context, uid string, id string) (*CustomProperty, error)

	// DeleteCustomProperty deletes the given custom property
	DeleteCustomProperty(ctx context.Context, uid string, id string) error

	// UpdateCustomProperty updates the value of the given customer property on the given device
	UpdateCustomProperty(ctx context.Context, uid string, id string, value string) error
}

type client struct {
	client   *http.Client
	url      string
	tid      int
	mTid     sync.Mutex
	username string
	password string
	monitor  string
}

const (
	actionDeviceRoute      action = "DeviceRouter"
	actionPropertiesRouter action = "PropertiesRouter"
	actionEventsRouter     action = "EventsRouter"

	// DeviceRouter methods https://help.zenoss.com/dev/collection-zone-and-resource-manager-apis/codebase/routers/router-reference/devicerouter
	methodGetDevices    method = "getDevices"
	methodAddDevice     method = "addDevice"
	methodRemoveDevices method = "removeDevices"
	methodSetInfo       method = "setInfo"

	// PropertiesRouter methods https://help.zenoss.com/dev/collection-zone-and-resource-manager-apis/codebase/routers/router-reference/propertiesrouter
	methodGetCustomProperties  method = "getCustomProperties" // Added method getCustomProperties to fetch custom properties
	methodUpdateCustomProperty method = "update"
	methodDeleteCustomProperty method = "remove"

	methodAddEvent method = "add_event"

	pathPropertiesRouter = "properties_router"
	pathDeviceRouter     = "device_router"
	pathEvconsoleRouter  = "evconsole_router"
)

// NewClient create new Zenoss instance
func NewClient(baseURI, username, password, monitor string, insecureSkipTLSVerify bool) (Client, error) {
	u, err := url.Parse(baseURI)
	if err != nil {
		return nil, fmt.Errorf("unable to parse given base URI: %w", err)
	}
	u.Path, _ = strings.CutSuffix(u.Path, "/zport/dmd")

	return &client{
		client: &http.Client{
			Transport: &http.Transport{
				//#nosec G402
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipTLSVerify},
			},
		},
		url:      u.String(),
		username: username,
		password: password,
		monitor:  monitor,
		tid:      0,
	}, nil
}

// AddEvent to component on device in Zenoss
func (api *client) AddEvent(ctx context.Context, summary, message, device, component string, severity Severity, evClass, evKey string, extraData map[string]string) error {
	if len(summary) > 255 {
		return fmt.Errorf("'summary' must be less than 255")
	}
	if len(message) > 4095 {
		return fmt.Errorf("'message' must be less than 4096 characters")
	}
	if len(component) > 255 {
		return fmt.Errorf("'component' must be less than 255")
	}
	if len(evKey) > 127 {
		return fmt.Errorf("'evKey' must be less than 127")
	}

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
		Action: actionEventsRouter,
		Method: methodAddEvent,
		Data:   []interface{}{d},
	}
	slog.Debug("Constructed request for Zenoss", "request", r)

	var resp addEventResponse
	err := api.doRequest(ctx, r, pathEvconsoleRouter, &resp)
	if err != nil {
		return fmt.Errorf("unable to create event: %w", err)
	}

	if !resp.Result.Success {
		return fmt.Errorf("event could not be created: %+v", resp)
	}

	return nil
}

func (z *client) ReadDevice(ctx context.Context, uid string) (*Device, error) {
	request := request{
		Action: actionDeviceRoute,
		Method: methodGetDevices,
		Data: []interface{}{
			deviceReadData{
				UID: uid,
			},
		},
	}
	var dev deviceReadResponse
	err := z.doRequest(ctx, request, pathDeviceRouter, &dev)
	if err != nil {
		return nil, fmt.Errorf("unable to read device: %w", err)
	}

	if dev.Result.Count > 1 {
		return nil, fmt.Errorf("error multiple devices returned")
	}

	if dev.Result.Count == 0 || len(dev.Result.Devices) == 0 {
		return nil, nil
	}

	if !dev.Result.Success {
		return nil, fmt.Errorf("error reading device")
	}

	return &dev.Result.Devices[0], nil
}

func (z *client) CreateDevice(ctx context.Context, dev NewDevice) (*Device, error) {
	req := request{
		Action: actionDeviceRoute,
		Method: methodAddDevice,
		Data: []interface{}{
			dev,
		},
	}
	var res deviceAddResponse
	err := z.doRequest(ctx, req, pathDeviceRouter, &res)
	if err != nil {
		return nil, fmt.Errorf("unable to create device: %w", err)
	}

	if !res.Result.Success {
		return nil, fmt.Errorf("add device returned unsuccessfull")
	}

	req = request{
		Action: actionDeviceRoute,
		Method: methodGetDevices,
		Data: []interface{}{
			deviceReadData{
				UID: "",
				Params: &deviceReadParams{
					Name: dev.Name,
				},
			},
		},
	}

	var read deviceReadResponse
	r := 0
	for {
		read = deviceReadResponse{}
		err = z.doRequest(ctx, req, pathDeviceRouter, &read)
		if err != nil {
			return nil, fmt.Errorf("unable to read device after creation: %w", err)
		}
		r++

		if !res.Result.Success {
			return nil, fmt.Errorf("read of added device returned unsuccessfull")
		}

		if read.Result.Count == 0 || len(read.Result.Devices) == 0 {
			if r > 5 { // 5* waiting 2 sec = 10 sec
				return nil, fmt.Errorf("no devices returned after creation of %s", dev.Name)
			}
		} else {
			break
		}

		select {
		case <-ctx.Done(): // Return if context is cancelled
			return nil, ctx.Err()
		case <-time.After(2 * time.Second): // Proceed after timeout
		}
	}

	if read.Result.Count > 1 || len(read.Result.Devices) > 1 {
		return nil, fmt.Errorf("multiple devices returned after creation of %s", dev.Name)
	}

	return &read.Result.Devices[0], nil
}

func (z *client) DeleteDevice(ctx context.Context, uid string) error {
	req := request{
		Action: actionDeviceRoute,
		Method: methodGetDevices,
		Data: []interface{}{
			deviceReadData{
				UID: uid,
			},
		},
	}
	var dev deviceReadResponse
	err := z.doRequest(ctx, req, pathDeviceRouter, &dev)
	if err != nil {
		return fmt.Errorf("unable to read hash for deleting device: %w", err)
	}

	req = request{
		Action: actionDeviceRoute,
		Method: methodRemoveDevices,
		Data: []interface{}{
			deviceRemoveData{
				Action:       "delete",
				UIDs:         []string{uid},
				Hashcheck:    dev.Result.Hash,
				DeleteEvents: true,
			},
		},
	}
	var res deviceRemoveResponse
	err = z.doRequest(ctx, req, pathDeviceRouter, &res)
	if err != nil {
		return fmt.Errorf("unable to delete device: %w", err)
	}

	if !res.Result.Success {
		return fmt.Errorf("remove device returned unsuccessful")
	}

	return nil
}

func (z *client) UpdateDeviceProductionState(ctx context.Context, uid string, state int) error {
	req := request{
		Action: actionDeviceRoute,
		Method: methodSetInfo,
		Data: []interface{}{
			deviceSetInfoData{
				UID:             uid,
				ProductionState: state,
			},
		},
	}
	var res deviceSetInfoResponse
	err := z.doRequest(ctx, req, pathDeviceRouter, &res)
	if err != nil {
		return fmt.Errorf("unable to update device: %w", err)
	}

	if !res.Result.Success {
		return fmt.Errorf("update device returned unsuccessful")
	}

	return nil
}

func (z *client) ReadCustomProperty(ctx context.Context, uid string, id string) (*CustomProperty, error) {
	request := request{
		Action: actionPropertiesRouter,
		Method: methodGetCustomProperties,
		Data: []interface{}{
			customPropertiesReadData{
				UID: uid,
				Params: &customPropReadParams{
					Id: id,
				},
			},
		},
	}
	var readResponse customPropertiesReadResponse
	err := z.doRequest(ctx, request, pathPropertiesRouter, &readResponse)
	if err != nil {
		return nil, fmt.Errorf("unable to read custom properties from device: %w", err)
	}

	if !readResponse.Result.Success || readResponse.Result.Count > 1 {
		return nil, fmt.Errorf("error reading custom properties from device or more devices returned")
	}

	if readResponse.Result.Count == 0 || len(readResponse.Result.Data) == 0 {
		return nil, nil
	}

	return &readResponse.Result.Data[0], nil
}

func (z *client) DeleteCustomProperty(ctx context.Context, uid string, id string) error {
	req := request{
		Action: actionPropertiesRouter,
		Method: methodGetCustomProperties,
		Data: []interface{}{
			customPropertiesReadData{
				UID: uid,
				Params: &customPropReadParams{
					Id: id,
				},
			},
		},
	}
	var prop customPropertiesReadResponse
	err := z.doRequest(ctx, req, pathPropertiesRouter, &prop)
	if err != nil {
		return fmt.Errorf("unable to read hash for deleting custom property: %w", err)
	}

	req = request{
		Action: actionPropertiesRouter,
		Method: methodDeleteCustomProperty,
		Data: []interface{}{
			customPropertyRemoveData{
				UID: uid,
				Id:  id,
			},
		},
	}
	var res customPropertyRemoveResponse
	err = z.doRequest(ctx, req, pathPropertiesRouter, &res)
	if err != nil {
		return fmt.Errorf("unable to delete custom property: %w", err)
	}

	if !res.Result.Success {
		return fmt.Errorf("remove custom property returned unsuccessful")
	}

	return nil
}

func (z *client) UpdateCustomProperty(ctx context.Context, uid string, id string, value string) error {
	req := request{
		Action: actionPropertiesRouter,
		Method: methodUpdateCustomProperty,
		Data: []interface{}{
			customPropertyUpdateData{
				UID:   uid,
				Id:    id,
				Value: value,
			},
		},
	}
	var res customPropertyUpdateResponse
	err := z.doRequest(ctx, req, pathPropertiesRouter, &res)
	if err != nil {
		return fmt.Errorf("unable to update custom property: %w", err)
	}

	if !res.Result.Success {
		return fmt.Errorf("update custom property returned unsuccessful")
	}

	return nil
}

func (z *client) doRequest(ctx context.Context, request request, routerPath string, target interface{}) error {
	request.Tid = z.nextTid()
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/zport/dmd/%s", z.url, routerPath), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("unable to create request for Zenoss: %w", err)
	}
	req.SetBasicAuth(z.username, z.password)
	req.Header.Set("Content-Type", "application/json")

	res, err := z.client.Do(req)
	if err != nil {
		return fmt.Errorf("error calling Zenoss: %s", err)
	}
	defer res.Body.Close()

	buf := &strings.Builder{}
	tee := io.TeeReader(res.Body, buf)
	err = json.NewDecoder(tee).Decode(target)
	if err != nil {
		return fmt.Errorf("unable to parse response from Zenoss: %w - response: %s", err, buf.String())
	}

	return nil
}

func (api *client) nextTid() int {
	api.mTid.Lock()
	defer api.mTid.Unlock()
	api.tid = api.tid + 1
	return api.tid
}
