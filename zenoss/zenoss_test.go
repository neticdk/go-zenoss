package zenoss

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/assert"
)

func TestAddEvent(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/zport/dmd/evconsole_router" {
			defer req.Body.Close()
			rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": true}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
		}
	}))
	defer server.Close()
	err := api.AddEvent(context.Background(), "summary", "message", "device", "component", SeverityCritical, "/Prometheus/KubeDeploymentReplicasMismatch", "key", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestAddEventExtraData(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/zport/dmd/evconsole_router" {
			defer req.Body.Close()
			rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": true}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
			var js map[string]interface{}
			json.NewDecoder(req.Body).Decode(&js)
			data := js["data"].([]interface{})[0].(map[string]interface{})
			assert.Equal(t, "/Prometheus/KubeDeploymentReplicasMismatch", data["evclass"])
			assert.Equal(t, "value1", data["prop1"])
			assert.Equal(t, "value2", data["prop2"])
		}
	}))
	defer server.Close()
	data := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
	}
	err := api.AddEvent(context.Background(), "summary", "message", "device", "component", SeverityCritical, "/Prometheus/KubeDeploymentReplicasMismatch", "key", data)
	if err != nil {
		t.Error(err)
	}
}

func TestAddEventError(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/zport/dmd/evconsole_router" {
			defer req.Body.Close()
			rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": false}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
		}
	}))
	defer server.Close()
	err := api.AddEvent(context.Background(), "summary", "message", "device", "component", SeverityCritical, "/Prometheus/KubeDeploymentReplicasMismatch", "key", nil)
	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestReadDevice(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/zport/dmd/device_router" {
			buf := new(strings.Builder)
			io.Copy(buf, req.Body)
			assert.Equal(t, `{"action":"DeviceRouter","method":"getDevices","data":[{"uid":"/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk"}],"tid":1}`, buf.String())
			assert.Equal(t, "POST", req.Method)
			rw.Write([]byte(readDeviceResponse))
		}
	}))
	defer server.Close()

	device, err := api.ReadDevice(context.Background(), "/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk")
	assert.NoError(t, err)
	assert.Equal(t, "oaas1.k8s.jysk.netic.dk", device.Name)
}

func TestCreateDevice(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "POST", req.Method)

		buf := new(strings.Builder)
		io.Copy(buf, req.Body)
		if strings.Contains(buf.String(), "addDevice") {
			assert.Equal(t, `{"action":"DeviceRouter","method":"addDevice","data":[{"deviceName":"oaas1.k8s.jysk.netic.dk","deviceClass":"/VirtualDevices/shared-kubernetes","collector":"localhost","model":true,"productionState":500,"groupPaths":["/SLA/Plus/Ping_Only"],"systemPaths":["/Netic/Test"],"locationPath":"/Netic/DC4"}],"tid":1}`, buf.String())
			rw.Write([]byte(addDeviceResponse))
		} else if strings.Contains(buf.String(), "\"tid\":2") {
			assert.Equal(t, `{"action":"DeviceRouter","method":"getDevices","data":[{"params":{"name":"oaas1.k8s.jysk.netic.dk"}}],"tid":2}`, buf.String())
			rw.Write([]byte(readDeviceResponseEmpty))
		} else {
			assert.Equal(t, `{"action":"DeviceRouter","method":"getDevices","data":[{"params":{"name":"oaas1.k8s.jysk.netic.dk"}}],"tid":3}`, buf.String())
			rw.Write([]byte(readDeviceResponse))
		}

	}))
	defer server.Close()

	dev := NewDevice{
		Name:            "oaas1.k8s.jysk.netic.dk",
		Class:           "/VirtualDevices/shared-kubernetes",
		Collector:       "localhost",
		Model:           true,
		ProductionState: 500,
		GroupPaths:      []string{"/SLA/Plus/Ping_Only"},
		SystemPaths:     []string{"/Netic/Test"},
		LocationPath:    "/Netic/DC4",
	}

	device, err := api.CreateDevice(context.Background(), dev)
	assert.NoError(t, err)
	assert.Equal(t, "oaas1.k8s.jysk.netic.dk", device.Name)
}

func TestDeleteDevice(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "POST", req.Method)

		buf := new(strings.Builder)
		io.Copy(buf, req.Body)
		if strings.Contains(buf.String(), "removeDevices") {
			assert.Equal(t, `{"action":"DeviceRouter","method":"removeDevices","data":[{"action":"delete","uids":["/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk"],"hashcheck":"1","deleteEvents":true}],"tid":2}`, buf.String())
			rw.Write([]byte(removeDeviceResponse))
		} else {
			assert.Equal(t, `{"action":"DeviceRouter","method":"getDevices","data":[{"uid":"/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk"}],"tid":1}`, buf.String())
			rw.Write([]byte(readDeviceResponse))
		}

	}))
	defer server.Close()

	err := api.DeleteDevice(context.Background(), "/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk")
	assert.NoError(t, err)
}

func TestUpdateDeviceProductionState(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		buf := new(strings.Builder)
		io.Copy(buf, req.Body)
		assert.Equal(t, `{"action":"DeviceRouter","method":"setInfo","data":[{"uid":"/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk","productionState":300}],"tid":1}`, buf.String())
		assert.Equal(t, "POST", req.Method)
		rw.Write([]byte(setInfoDeviceResponse))
	}))
	defer server.Close()

	err := api.UpdateDeviceProductionState(context.Background(), "/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk", 300)
	assert.NoError(t, err)
}

func TestReadCustomProperty(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		buf := new(strings.Builder)
		io.Copy(buf, req.Body)
		assert.Equal(t, `{"action":"PropertiesRouter","method":"getCustomProperties","data":[{"uid":"/zport/dmd/Devices/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk","params":{"id":"cValue"}}],"tid":1}`, buf.String())
		assert.Equal(t, "POST", req.Method)
		rw.Write([]byte(readCustomPropertyResponse))
	}))
	defer server.Close()

	prop, err := api.ReadCustomProperty(context.Background(), "/zport/dmd/Devices/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk", "cValue")
	assert.NoError(t, err)
	assert.Equal(t, "cValue", prop.Id)
	assert.Equal(t, "15", prop.Value)
}

func TestUpdateCustomProperty(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		buf := new(strings.Builder)
		io.Copy(buf, req.Body)
		assert.Equal(t, `{"action":"PropertiesRouter","method":"update","data":[{"uid":"/zport/dmd/Devices/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk","id":"cValue","value":"15"}],"tid":1}`, buf.String())
		assert.Equal(t, "POST", req.Method)
		rw.Write([]byte(updateCustomPropertyResponse))
	}))
	defer server.Close()

	err := api.UpdateCustomProperty(context.Background(), "/zport/dmd/Devices/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk", "cValue", "15")
	assert.NoError(t, err)
}

func TestDeleteCustomProperty(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "POST", req.Method)

		buf := new(strings.Builder)
		io.Copy(buf, req.Body)
		if strings.Contains(buf.String(), "remove") {
			assert.Equal(t, `{"action":"PropertiesRouter","method":"remove","data":[{"uid":"/zport/dmd/Devices/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk","id":"cValue"}],"tid":2}`, buf.String())
			rw.Write([]byte(deleteCustomPropertyResponse))
		} else {
			assert.Equal(t, `{"action":"PropertiesRouter","method":"getCustomProperties","data":[{"uid":"/zport/dmd/Devices/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk","params":{"id":"cValue"}}],"tid":1}`, buf.String())
			rw.Write([]byte(readCustomPropertyResponse))
		}

	}))
	defer server.Close()

	err := api.DeleteCustomProperty(context.Background(), "/zport/dmd/Devices/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk", "cValue")
	assert.NoError(t, err)
}

func newStubAPI(handler http.HandlerFunc) (Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	api := &client{
		client: server.Client(),
		url:    server.URL,
	}
	return api, server
}

func init() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
}

const readDeviceResponse = `{
	"uuid": "f353184f-59f4-4057-9cc6-8614dd7cc91e",
	"action": "DeviceRouter",
	"result": {
	  "totalCount": 1,
	  "hash": "1",
	  "success": true,
	  "devices": [
		{
		  "ipAddressString": "10.238.84.99",
		  "serialNumber": "",
		  "pythonClass": "Products.ZenModel.Device",
		  "hwManufacturer": null,
		  "collector": "localhost",
		  "osModel": null,
		  "productionState": 1000,
		  "systems": [
			{
			  "uid": "/zport/dmd/Systems/Jysk/Development",
			  "path": "/Systems/Jysk/Development",
			  "uuid": "3c4b7db9-b102-45ea-ade9-62ebc2532ac1",
			  "name": "/Jysk/Development"
			}
		  ],
		  "priority": 3,
		  "hwModel": null,
		  "tagNumber": "",
		  "osManufacturer": null,
		  "location": {
			"uid": "/zport/dmd/Locations/Netic",
			"name": "/Netic",
			"uuid": "b6d0828f-f646-4666-b327-512071dc6e7a"
		  },
		  "groups": [
			{
			  "uid": "/zport/dmd/Groups/SLA/Standard",
			  "path": "/Groups/SLA/Standard",
			  "uuid": "2897c28f-f34a-4444-a353-ace91221bc74",
			  "name": "/SLA/Standard"
			},
			{
			  "uid": "/zport/dmd/Groups/JiraSLA/AppDriftPlus",
			  "path": "/Groups/JiraSLA/AppDriftPlus",
			  "uuid": "b5b4399a-1c1d-4658-9aa9-cb05305ed313",
			  "name": "/JiraSLA/AppDriftPlus"
			}
		  ],
		  "uid": "/zport/dmd/Devices/VirtualDevices/jysk-k8s/devices/oaas1.k8s.jysk.netic.dk",
		  "ipAddress": 183391331,
		  "events": {
			"info": {
			  "count": 0,
			  "acknowledged_count": 0
			},
			"clear": {
			  "count": 0,
			  "acknowledged_count": 0
			},
			"warning": {
			  "count": 0,
			  "acknowledged_count": 0
			},
			"critical": {
			  "count": 0,
			  "acknowledged_count": 0
			},
			"error": {
			  "count": 0,
			  "acknowledged_count": 0
			},
			"debug": {
			  "count": 0,
			  "acknowledged_count": 0
			}
		  },
		  "name": "oaas1.k8s.jysk.netic.dk"
		}
	  ]
	},
	"tid": 1,
	"type": "rpc",
	"method": "getDevices"
  }`

const readDeviceResponseEmpty = `{
	"uuid": "b2033e78-0cb5-42b7-878e-062386b777c7",
	"action": "DeviceRouter",
	"result": {
	  "totalCount": 0,
	  "hash": "0",
	  "success": true,
	  "devices": []
	},
	"tid": 1,
	"type": "rpc",
	"method": "getDevices"
  }`

const addDeviceResponse = `{
	"uuid": "b55a545a-c470-4a34-88a2-7e1c69190008",
	"action": "DeviceRouter",
	"result": {
	  "new_jobs": [
		{
		  "uuid": "c3ab77cf-fdea-4daa-8c3e-7131049439d1",
		  "description": "Create prod1.netic-platform.shared.k8s.netic.dk under /VirtualDevices/shared-kubernetes",
		  "uid": "/zport/dmd/JobManager"
		},
		{
		  "uuid": "fe2a1578-0d89-4370-ab0d-d4cc86d1ca73",
		  "description": "Discover and model device prod1.netic-platform.shared.k8s.netic.dk as /VirtualDevices/shared-kubernetes",
		  "uid": "/zport/dmd/JobManager"
		}
	  ],
	  "success": true
	},
	"tid": 1,
	"type": "rpc",
	"method": "addDevice"
  }`

const removeDeviceResponse = `{
	"uuid": "f4041b9a-e41d-4cf6-b4f4-aef1f53c231e",
	"action": "DeviceRouter",
	"result": {
	  "success": true
	},
	"tid": 1,
	"type": "rpc",
	"method": "removeDevices"
  }`

const setInfoDeviceResponse = `{
	"uuid": "6fd92cf9-1477-4548-921f-c3290e7519ce",
	"action": "DeviceRouter",
	"result": {
	  "success": true
	},
	"tid": 1,
	"type": "rpc",
	"method": "setInfo"
  }`

const readCustomPropertyResponse = `{
	"uuid": "d200dcf2-542e-49c2-be17-d2dfcbe9cf71",
	"action": "PropertiesRouter",
	"result": {
	  "totalCount": 1,
	  "data": [
		{
		  "islocal": 1,
		  "value": "15",
		  "label": "Custom Value",
		  "valueAsString": "15",
		  "id": "cValue",
		  "path": "/VirtualDevices/shared-kubernetes/devices/prod1.netic-platform.shared.k8s.netic.dk",
		  "type": "string",
		  "options": []
		}
	  ],
	  "success": true
	},
	"tid": 1,
	"type": "rpc",
	"method": "getCustomProperties"
  }`

const updateCustomPropertyResponse = `{
	"uuid": "27bd2f0b-f865-4c3e-9268-0f0ae2053215",
	"action": "PropertiesRouter",
	"result": {
	  "msg": "Property cValue successfully updated.",
	  "success": true
	},
	"tid": 1,
	"type": "rpc",
	"method": "update"
  }`

const deleteCustomPropertyResponse = `{
	"uuid": "4449fe52-3cc2-4099-88a0-8caafa40f5b3",
	"action": "PropertiesRouter",
	"result": {
	  "msg": "Property cValue successfully deleted.",
	  "success": true
	},
	"tid": 2,
	"type": "rpc",
	"method": "remove"
  }`
