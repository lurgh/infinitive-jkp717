package main

import (
	"fmt"
	"time"
	"strings"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type EventListener struct {
	ch chan []byte
}

type discoveryTopic struct {
	Topic       string    `json:"state_topic"`
	Name        string    `json:"name"`
	Device_class string   `json:"device_class,omitempty"`
	UoM         string   `json:"unit_of_measurement,omitempty"`
	Unique_id   string    `json:"unique_id"`
}

type EventDispatcher struct {
	listeners  map[*EventListener]bool
	broadcast  chan []byte
	register   chan *EventListener
	deregister chan *EventListener
}

type MqttEvent struct {
	topic string
	value string
}

type MqttConn struct {
	url string	// "tcp://host.com:1883"
	password string
}

var Dispatcher *EventDispatcher = newEventDispatcher()

var mqttClient mqtt.Client

func newEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		broadcast:  make(chan []byte, 64),
		register:   make(chan *EventListener),
		deregister: make(chan *EventListener),
		listeners:  make(map[*EventListener]bool),
	}
}

func (d *EventDispatcher) dispatch(msg []byte) {
	d.broadcast <- msg
}

type broadcastEvent struct {
	Source string      `json:"source"`
	Data   interface{} `json:"data"`
}

func serializeEvent(source string, data interface{}) []byte {
	msg, _ := json.Marshal(&broadcastEvent{Source: source, Data: data})
	return msg
}

func (d *EventDispatcher) broadcastEvent(source string, data interface{}) {
	if source[0:5] == "mqtt/" {
		if mqttClient != nil {
			topic := source[5:]
			value := fmt.Sprintf("%v", data)
			log.Infof("MQTT PUB: %s -> %s", topic, value)
			_ = mqttClient.Publish(topic, 0, true, value)
		}
	} else {
		d.broadcast <- serializeEvent(source, data)
	}
}

func (h *EventDispatcher) run() {
	for {
		select {
		case listener := <-h.register:
			h.listeners[listener] = true
		case listener := <-h.deregister:
			if _, ok := h.listeners[listener]; ok {
				delete(h.listeners, listener)
				close(listener.ch)
			}
		case message := <-h.broadcast:
			for listener := range h.listeners {
				select {
				case listener.ch <- message:
				default:
					close(listener.ch)
					delete(h.listeners, listener)
				}
			}
		}
	}
}

// handle messages
// topics: infinitive/SETTING/set (global)
//	infinitive/zone/X/SETTING/set (zone X)
func  mqttMessageHandler(client mqtt.Client, msg mqtt.Message) {
	log.Infof("MQTT: Received message: %s from topic: %s", msg.Payload(), msg.Topic())

	ts := strings.Split(msg.Topic(), "/")
	ps := fmt.Sprintf("%s", msg.Payload())

	if len(ts) < 3 || ts[0] != "infinitive" || ts[len(ts)-1] != "set" {
		log.Errorf("mqtt received unexpected topic '%s'", msg.Topic())
	} else if len(ts) == 5 && ts[1] == "zone" {
		// zone-based
		if ps[len(ps)-2:len(ps)-1] == "." {
			ps = ps[0:len(ps)-2]
		}
		_ = putConfig(ts[2], ts[3], ps)
	} else if len(ts) == 4 && ts[1] == "vacation" {
		_ = putVacationConfig(ts[2], ps)
	} else if len(ts) == 3 {
		// global
		_ = putConfig("0", ts[1], ps)
	} else {
		log.Errorf("mqtt received malformed topic '%s'", msg.Topic())
	}
}

// set up for async connect/reconnect (for robustness across restarts on eithesride) 
func ConnectMqtt(url string, password string) {

	// set mqtt client options
	co := mqtt.NewClientOptions()
	co.AddBroker(url)
	co.SetPassword(password)
	co.SetClientID("infinitive_mqtt_client")
	co.SetOnConnectHandler(mqttOnConnect)
	co.SetConnectionLostHandler(func(cl mqtt.Client, err error) {log.Info("MQTT: Connection lost: ", err.Error())})
	co.SetReconnectingHandler(func(cl mqtt.Client, _ *mqtt.ClientOptions) {log.Info("MQTT: Trying to reconnect")})
	co.SetConnectRetry(true)
	co.SetConnectRetryInterval(time.Minute)
	co.SetAutoReconnect(true)
	co.SetMaxReconnectInterval(5 * time.Minute)

	// create client
	mqttClient = mqtt.NewClient(co)

	// start trying to connect - resolved in callbacks
	log.Info("MTQQ: Start trying to connect")
	mqttClient.Connect()
}

// on connect, subscribe to needed topics
func mqttOnConnect(cl mqtt.Client) {
	log.Info("MQTT: Connected, subscribing...")

	// subscribe for zone settings
	t := cl.Subscribe("infinitive/zone/+/+/set", 0, mqttMessageHandler)
	t.Wait()
	if (t.Error() != nil) {
		log.Error("MQTT: failed to subscribe for infinitive/zone/+/+/set: ", t.Error())
	} else {
		log.Info("MQTT: subscribe succeeded for infinitive/zone/+/+/set")
	}

	// subscribe for vacation settings
	t = cl.Subscribe("infinitive/vacation/+/set", 0, mqttMessageHandler)
	t.Wait()
	if (t.Error() != nil) {
		log.Error("MQTT: failed to subscribe for infinitive/vacation/+/set: ", t.Error())
	} else {
		log.Info("MQTT: subscribe succeeded for infinitive/vacation/+/set")
	}

	// subscribe for global settings
	t = cl.Subscribe("infinitive/+/set", 0, mqttMessageHandler)
	t.Wait()
	if (t.Error() != nil) {
		log.Error("MQTT: failed to subscribe for infinitive/+/set: ", t.Error())
	} else {
		log.Info("MQTT: subscribe succeeded for infinitive/+/set")
	}

	discoveryTopics := []discoveryTopic {
		{ "infinitive/outdoorTemp", "HVAC Outdoor Temperature", "temperature", "°F", "hvac-sensors-odt" },
		{ "infinitive/humidity", "HVAC Indoor Humidity", "humidity", "%", "hvac-sensors-hum" },
		{ "infinitive/rawMode", "HVAC Raw Mode", "", "", "hvac-sensors-rawmode" },
		{ "infinitive/blowerRPM", "HVAC Blower RPM", "", "RPM", "hvac-sensors-blowerrpm" },
		{ "infinitive/airflowCFM", "HVAC Airflow CFM", "", "CFM", "hvac-sensors-aflo" },
		{ "infinitive/staticPressure", "HVAC Static Pressure", "distance", "in", "hvac-sensors-ahsp" },
		{ "infinitive/coolStage", "HVAC Cool Stage", "", "", "hvac-sensors-acstage" },
		{ "infinitive/heatStage", "HVAC Heat Stage", "", "", "hvac-sensors-heatstage" },
		{ "infinitive/action", "HVAC Action", "enum", "", "hvac-sensors-actn" },

		{ "infinitive/vacation/active", "Vacation Mode Active", "enum", "", "hvac-sensors-vacay-active" },  // maybe should be a binary_sensor
		{ "infinitive/vacation/days", "Vacation Mode Days Remaining", "duration", "d", "hvac-sensors-vacay-days" },
		{ "infinitive/vacation/hours", "Vacation Mode Hours Remaining", "duration", "h", "hvac-sensors-vacay-hours" },
		{ "infinitive/vacation/minTemp", "Vacation Mode Minimum Temperature", "temperature", "°F", "hvac-sensors-vacay-mint" },
		{ "infinitive/vacation/maxTemp", "Vacation Mode Maximum Temperature", "temperature", "°F", "hvac-sensors-vacay-maxt" },
		{ "infinitive/vacation/minHumidity", "Vacation Mode Minimum Humidity", "humidity", "%", "hvac-sensors-vacay-minh" },
		{ "infinitive/vacation/maxHumidity", "Vacation Mode Maximum Humidity", "humidity", "%", "hvac-sensors-vacay-maxh" },
		{ "infinitive/vacation/fanMode", "Vacation Mode Fan Mode", "enum", "", "hvac-sensors-vacay-fm" },

		// per-zone "bonus" sensors (outside of the Climate platform model)
		// TODO: these should be parametrized, maybe do along with the Climate entities)
		{ "infinitive/zone/1/damperPos", "HVAC Zone 1 Damper Postion", "", "%", "hvac-sensors-z1-dpos" },
		{ "infinitive/zone/2/damperPos", "HVAC Zone 2 Damper Postion", "", "%", "hvac-sensors-z2-dpos" },
		{ "infinitive/zone/1/flowWeight", "HVAC Zone 1 Airflow Weight", "", "", "hvac-sensors-z1-fwgt" },
		{ "infinitive/zone/2/flowWeight", "HVAC Zone 2 Airflow Weight", "", "", "hvac-sensors-z2-fwgt" },
		{ "infinitive/zone/1/overrideDurationMins", "HVAC Zone 1 Override Duration", "duration", "min", "hvac-sensors-z1-odur" },
		{ "infinitive/zone/2/overrideDurationMins", "HVAC Zone 2 Override Duration", "duration", "min", "hvac-sensors-z2-odur" },
	}

	// write discovery topics for HA
	/*
	_ = cl.Publish("homeassistant/sensor/infinitive/hs/config", 0, true,
		`{"state_topic": "infinitive/heatStage","state_class": "measurement",
		"name": "Heat Stage",
		"unique_id": "hvac-sensors-heatstage"}`)
		*/
	for _, v := range discoveryTopics {
		log.Errorf("MQTT STR %v", &v)
		j, err := json.Marshal(&v)
		log.Errorf("MQTT PUB %v: %s", err, j)
		if err == nil {
			_ = cl.Publish("homeassistant/sensor/infinitive/" + v.Unique_id + "/config", 0, true, j)
		}
	}

	// flush the MQTT value cache
	mqttCache.clear()
}

func init() {
	go Dispatcher.run()
}
