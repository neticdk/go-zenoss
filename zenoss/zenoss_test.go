package zenoss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddEvent(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": true}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
	}))
	defer server.Close()
	err := api.AddEvent(context.Background(), "summary", "message", "device", "component", SeverityCritical, "/Prometheus/KubeDeploymentReplicasMismatch", "key")
	if err != nil {
		t.Error(err)
	}
}

func TestAddEventError(t *testing.T) {
	api, server := newStubAPI(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{"uuid": "369b07eb-3b90-46bc-afb9-6c75bf53e3ea", "action": "EventsRouter", "result": {"msg": "Created event", "success": false}, "tid": 1589956412, "type": "rpc", "method": "add_event"}`))
	}))
	defer server.Close()
	err := api.AddEvent(context.Background(), "summary", "message", "device", "component", SeverityCritical, "/Prometheus/KubeDeploymentReplicasMismatch", "key")
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
