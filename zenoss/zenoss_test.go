package zenoss

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddEvent(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": true}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
	}))
	defer server.Close()
	err := api.AddEvent(context.Background(), "summary", "message", "device", "component", SeverityCritical, "/Prometheus/KubeDeploymentReplicasMismatch", "key", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestAddEventExtraData(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": true}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
		var js map[string]interface{}
		json.NewDecoder(req.Body).Decode(&js)
		data := js["data"].([]interface{})[0].(map[string]interface{})
		assert.Equal(t, "/Prometheus/KubeDeploymentReplicasMismatch", data["evclass"])
		assert.Equal(t, "value1", data["prop1"])
		assert.Equal(t, "value2", data["prop2"])
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
		defer req.Body.Close()
		rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": false}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
	}))
	defer server.Close()
	err := api.AddEvent(context.Background(), "summary", "message", "device", "component", SeverityCritical, "/Prometheus/KubeDeploymentReplicasMismatch", "key", nil)
	if err == nil {
		t.Errorf("Expected error")
	}
}

func newStubAPI(handler http.HandlerFunc) (Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	api := &client{
		client:  server.Client(),
		baseURI: server.URL,
	}
	return api, server
}
