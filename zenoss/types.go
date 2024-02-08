package zenoss

import "fmt"

// Severity defines event severity
type Severity string

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

type Device struct {
	UID             string `json:"uid"`
	Name            string `json:"name"`
	ProductionState int    `json:"productionState"`
}

type NewDevice struct {
	Name            string   `json:"deviceName"`
	Class           string   `json:"deviceClass"`
	Collector       string   `json:"collector"`
	Model           bool     `json:"model"`
	ProductionState int      `json:"productionState"`
	GroupPaths      []string `json:"groupPaths"`
	SystemPaths     []string `json:"systemPaths"`
	LocationPath    string   `json:"locationPath,omitempty"`
}

type CustomProperty struct {
	UID     string `json:"uid,omitempty"`
	Id      string `json:"id"`
	Value   string `json:"value"`
	IsLocal int    `json:"islocal"`
}

type deviceReadData struct {
	UID    string            `json:"uid,omitempty"`
	Params *deviceReadParams `json:"params,omitempty"`
}

// Can be one of the following: name, ipAddress, deviceClass, or productionState
type deviceReadParams struct {
	Name string `json:"name,omitempty"`
}

type action string
type method string

type request struct {
	Action action        `json:"action"`
	Method method        `json:"method"`
	Data   []interface{} `json:"data"`
	Tid    int           `json:"tid"`
}

func (r request) String() string {
	return fmt.Sprintf("{action=%s,method=%s,data=%v,tid=%d}", r.Action, r.Method, r.Data, r.Tid)
}

type response struct {
	UUID   string `json:"uuid"`
	Action action `json:"action"`
	Method method `json:"method"`
	Tid    int    `json:"tid"`
}

func (r response) String() string {
	return fmt.Sprintf("{uuid=%s,action=%s,method=%s,tid=%d}", r.UUID, r.Action, r.Method, r.Tid)
}

type result struct {
	Success bool `json:"success"`
}

type addEventResponse struct {
	response
	Result result `json:"result"`
}

type deviceReadResponse struct {
	response
	Result deviceReadResult `json:"result"`
}

type deviceReadResult struct {
	result
	Count   int      `json:"totalCount"`
	Hash    string   `json:"hash"`
	Devices []Device `json:"devices"`
}

type deviceAddResponse struct {
	response
	Result deviceAddResult `json:"result"`
}

type deviceAddResult struct {
	result
	Jobs []deviceAddJob `json:"new_jobs"`
}

type deviceAddJob struct {
	UUID string `json:"uuid"`
}

type deviceRemoveData struct {
	Action       string   `json:"action"`
	UIDs         []string `json:"uids"`
	Hashcheck    string   `json:"hashcheck"`
	DeleteEvents bool     `json:"deleteEvents"`
}

type deviceRemoveResponse struct {
	response
	Result result `json:"result"`
}

type deviceSetInfoData struct {
	UID             string `json:"uid"`
	ProductionState int    `json:"productionState"`
}

type deviceSetInfoResponse struct {
	response
	Result result `json:"result"`
}

type customPropertiesReadData struct {
	UID    string                `json:"uid,omitempty"`
	Params *customPropReadParams `json:"params,omitempty"`
}

type customPropReadParams struct {
	Id string `json:"id,omitempty"`
}

type customPropertiesReadResponse struct {
	response
	Result customPropertiesReadResult `json:"result"`
}

type customPropertiesReadResult struct {
	result
	Count int              `json:"totalCount"`
	Data  []CustomProperty `json:"data"`
}

type customPropertyRemoveData struct {
	UID string `json:"uid"`
	Id  string `json:"id"`
}

type customPropertyRemoveResponse struct {
	response
	Result result `json:"result"`
}

type customPropertyUpdateData struct {
	UID   string `json:"uid"`
	Id    string `json:"id"`
	Value string `json:"value"`
}

type customPropertyUpdateResponse struct {
	response
	Result result `json:"result"`
}
