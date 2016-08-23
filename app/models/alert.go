package models

import (
	"github.com/twinj/uuid"
	"time"
)

//http://docs.alerta.io/en/latest/api/alert.html
type Alert struct {
	//----------------------------Attributes generated by source section ---------------------------

	//globally unique random UUID
	Id string `gorethink:"id,omitempty" json:"id"`

	//resource under alarm, deliberately not host-centric
	Resource string `gorethink:"resource" json:"resource"`

	//event name eg. NodeDown, QUEUE:LENGTH:EXCEEDED, AppStatus
	Event string `gorethink:"event" json:"event"`

	//effected environment, used to namespace the defined resource
	Environment string `gorethink:"environment" json:"environment"`

	//severity of alert (default normal). see Alert Severities table
	Severity string `gorethink:"severity" json:"severity"`

	//list of related event names
	Correlate []string `gorethink:"correlate" json:"correlate"`

	//status of alert (default open). see Alert Status table
	Status string `gorethink:"status" json:"status"`

	//list of effected services
	Service []string `gorethink:"service" json:"service"`

	//event group
	Group string `gorethink:"group" json:"group"`

	//event value eg. 100%, Down, PingFail, 55ms, ORA-1664
	Value string `gorethink:"value" json:"value"`

	//freeform text description
	Text string `gorethink:"text" json:"text"`

	//set of tags in any format eg. aTag, aDouble:Tag, a:Triple=Tag
	Tags []string `gorethink:"tags" json:"tags"`

	//dictionary of key-value pairs
	Attributes map[string]string `gorethink:"attributes" json:"attributes"`

	//name of monitoring component that generated the alert
	Origin string `gorethink:"origin" json:"origin"`

	//alert type eg. snmptrapAlert, syslogAlert, gangliaAlert
	EventType string `gorethink:"eventType" json:"eventType"`

	//UTC date and time the alert was generated in ISO 8601 format
	CreateTime time.Time `gorethink:"createTime" json:"createTime"`

	//number of seconds before alert is considered stale
	Timeout int `gorethink:"timeout" json:"timeout"`

	//number of seconds before alert status is moved from ack -> open
	AcknowledgementDuration int `gorethink:"acknowledgement_duration" json:"acknowledgement_duration"`

	//unprocessed data eg. full syslog message or SNMP trap
	RawData string `gorethink:"rawData" json:"rawData"`

	//Extra namespace for tracking customers in the same enviornment
	Customer string `gorethink:"customer" json:"customer"`

	//-------------------------------Attributes created at processing time---------------------------------

	//a count of the number of times this event has been received for a resource
	DuplicateCount int `gorethink:"duplicateCount" json:"duplicateCount"`

	//if duplicateCount is 0 or the alert status has changed then repeat is False, otherwise it is True
	Repeat bool `gorethink:"repeat" json:"repeat"`

	//the previous severity of the same event for this resource. if no event or correlate events exist in the database for this resource then it will be unknown
	PreviousSeverity string `gorethink:"previousSeverity" json:"previousSeverity"`

	//based on severity and previousSeverity will be one of moreSevere, lessSevere or noChange
	TrendIndication string `gorethink:"trendIndication" json:"trendIndication"`

	//UTC date and time the alert was received by the Alerta server daemon
	ReceiveTime time.Time `gorethink:"receiveTime" json:"receiveTime"`

	//the last alert id received for this event
	LastReceiveId string `gorethink:"lastReceiveId" json:"lastReceiveId"`

	//the last time this alert was received. only different to receiveTime if the alert is a duplicate
	LastReceiveTime time.Time `gorethink:"lastReceiveTime" json:"lastReceiveTime"`

	//whenever an alert changes severity or status then a list of key alert attributes are appended to the history log
	History []HistoryEvent `gorethink:"history" json:"history"`
}

type HistoryEvent struct {
	Id         string    `gorethink:"id,omitempty" json:"id"`
	Event      string    `gorethink:"event" json:"event"`
	Status     string    `gorethink:"status" json:"status"`
	Severity   string    `gorethink:"severity" json:"severity"`
	Value      string    `gorethink:"value" json:"value"`
	Type       string    `gorethink:"type" json:"type"`
	Text       string    `gorethink:"text" json:"text"`
	UpdateTime time.Time `gorethink:"updateTime" json:"updateTime"`
}

func (alert *Alert) GenerateDefaults() {
	if alert.Id == "" {
		id := uuid.NewV4()
		alert.Id = id.String()
	}

	if alert.Status == "" {
		alert.Status = "open"
	}

	if alert.Attributes == nil {
		alert.Attributes = map[string]string{}
	}

	if alert.CreateTime.IsZero() {
		alert.CreateTime = time.Now()
	}

	if alert.ReceiveTime.IsZero() {
		alert.ReceiveTime = time.Now()
	}

	if alert.LastReceiveTime.IsZero() {
		alert.LastReceiveTime = time.Now()
		alert.LastReceiveId = alert.Id
	}

	alert.History = []HistoryEvent{HistoryEvent{
		Id:         alert.Id,
		Event:      alert.Event,
		Severity:   alert.Severity,
		Value:      alert.Value,
		Type:       alert.EventType,
		Text:       alert.Text,
		UpdateTime: alert.CreateTime,
	}}
}

type AlertResponse struct {
	Status string `json:"status"`
	Total  int    `json:"total"`
	Alert  Alert  `json:"alert"`
}

func NewAlertResponse(alert Alert) (a AlertResponse) {
	a = AlertResponse{}
	a.Alert = alert
	a.Total = 1
	a.Status = "ok"

	return
}

type AlertsResponse struct {
	Status         string                   `json:"status"`
	Total          int                      `json:"total"`
	Alerts         []map[string]interface{} `json:"alerts"`
	Page           int                      `json:"page"`
	PageSize       int                      `json:"pageSize"`
	Pages          int                      `json:"pages"`
	More           bool                     `json:"more"`
	SeverityCounts int                      `json:"severityCounts"`
	StatusCounts   int                      `json:"statusCounts"`
	LastTime       time.Time                `json:"lastTime"`
	AutoRefresh    bool                     `json:"autoRefresh"`
}

func NewAlertsResponse(alerts []map[string]interface{}) (ar AlertsResponse) {
	ar = AlertsResponse{}
	ar.Alerts = alerts
	ar.Total = len(alerts)
	ar.Status = "ok"
	ar.AutoRefresh = true
	if ar.Total > 0 {
		ar.LastTime = time.Now()
	}
	ar.More = false
	ar.Page = 1
	ar.Pages = 1
	ar.PageSize = 10000

	return
}

type AlertChangeFeed struct {
	OldVal Alert  `gorethink:"old_val"`
	NewVal Alert  `gorethink:"new_val"`
	Type   string `gorethink:"type"`
}
