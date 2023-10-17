package main

import (
	"fmt"
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
	State_class string    `json:"state_class"`
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

func ConnectMqtt(url string, password string) {

	// set mqtt client options
	co := mqtt.NewClientOptions()
	co.AddBroker(url)
	co.SetPassword(password)
	co.SetClientID("infinitive_mqtt_client")

	// create client
	cl := mqtt.NewClient(co)

	// connect
	t := cl.Connect()
	t.Wait()
	if (t.Error() != nil) {
		log.Error("MQTT: failed to connect to MQTT broker: ", t.Error())
	} else {
		log.Info("MQTT: connected to MQTT broker")
		mqttClient = cl
	}

	// subscribe for zone settings
	t = cl.Subscribe("infinitive/zone/+/+/set", 0, mqttMessageHandler)
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
		{ "infinitive/heatStage", "Heat Stage", "measurement", "hvac-sensors-heatstage" },
		{ "infinitive/vacation/active", "Vacation Mode Active", "measurement", "hvac-sensors-vacay-active" },
		{ "infinitive/vacation/days", "Vacation Mode Days Remaining", "measurement", "hvac-sensors-vacay-days" },
		{ "infinitive/vacation/hours", "Vacation Mode Hours Remaining", "measurement", "hvac-sensors-vacay-hours" },
		{ "infinitive/vacation/minTemp", "Vacation Mode Minimum Temperature", "measurement", "hvac-sensors-vacay-mint" },
		{ "infinitive/vacation/maxTemp", "Vacation Mode Maximum Temperature", "measurement", "hvac-sensors-vacay-maxt" },
		{ "infinitive/vacation/minHumidity", "Vacation Mode Minimum Humidity", "measurement", "hvac-sensors-vacay-minh" },
		{ "infinitive/vacation/maxHumidity", "Vacation Mode Maximum Humidity", "measurement", "hvac-sensors-vacay-maxh" },
		{ "infinitive/vacation/fanMode", "Vacation Mode Fan Mode", "measurement", "hvac-sensors-vacay-fm" },
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
}

func init() {
	go Dispatcher.run()
}
