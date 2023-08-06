package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

type TStatZone0Config struct {
	CurrentTempZ1     uint8  `json:"currentTempZ1"`
	CurrentHumidityZ1 uint8  `json:"currentHumidityZ1"`
	CurrentTempZ2     uint8  `json:"currentTempZ2"`
	CurrentHumidityZ2 uint8  `json:"currentHumidityZ2"`
	CurrentTempZ3     uint8  `json:"currentTempZ3"`
	CurrentHumidityZ3 uint8  `json:"currentHumidityZ3"`
	CurrentTempZ4     uint8  `json:"currentTempZ4"`
	CurrentHumidityZ4 uint8  `json:"currentHumidityZ4"`
	OutdoorTemp       uint8  `json:"outdoorTemp"`
	Mode              string `json:"mode"`
	Stage             uint8  `json:"stage"`
	FanModeZ1         string `json:"fanModeZ1"`
	FanModeZ2         string `json:"fanModeZ2"`
	FanModeZ3         string `json:"fanModeZ3"`
	FanModeZ4         string `json:"fanModeZ4"`
	HoldZ1            *bool  `json:"holdZ1"`
	HoldZ2            *bool  `json:"holdZ2"`
	HoldZ3            *bool  `json:"holdZ3"`
	HoldZ4            *bool  `json:"holdZ4"`
	HeatSetpointZ1    uint8  `json:"heatSetpointZ1"`
	CoolSetpointZ1    uint8  `json:"coolSetpointZ1"`
	HeatSetpointZ2    uint8  `json:"heatSetpointZ2"`
	CoolSetpointZ2    uint8  `json:"coolSetpointZ2"`
	HeatSetpointZ3    uint8  `json:"heatSetpointZ3"`
	CoolSetpointZ3    uint8  `json:"coolSetpointZ3"`
	HeatSetpointZ4    uint8  `json:"heatSetpointZ4"`
	CoolSetpointZ4    uint8  `json:"coolSetpointZ4"`
	HoldDurationZ1    string `json:"holdDurationZ1"`
	HoldDurationZ2    string `json:"holdDurationZ2"`
	HoldDurationZ3    string `json:"holdDurationZ3"`
	HoldDurationZ4    string `json:"holdDurationZ4"`
	ZoneNameZ1        string `json:"zoneNameZ1"`
	ZoneNameZ2        string `json:"zoneNameZ2"`
	ZoneNameZ3        string `json:"zoneNameZ3"`
	ZoneNameZ4        string `json:"zoneNameZ4"`
	RawMode           uint8  `json:"rawMode"`
}

type TStatZoneConfig struct {
	CurrentTemp     uint8  `json:"currentTemp"`
	CurrentHumidity uint8  `json:"currentHumidity"`
	OutdoorTemp     uint8  `json:"outdoorTemp"`
	Mode            string `json:"mode"`
	Stage           uint8  `json:"stage"`
	FanMode         string `json:"fanMode"`
	Hold            *bool  `json:"hold"`
	HeatSetpoint    uint8  `json:"heatSetpoint"`
	CoolSetpoint    uint8  `json:"coolSetpoint"`
	HoldDuration	string `json:"holdDuration"`
	HoldDurationMins uint16 `json:"holdDurationMins"`
	TargetHumidity  uint8  `json:"targetHumidity"`
	ZoneName	string `json:"zoneName"`
	RawMode         uint8  `json:"rawMode"`
}

type AirHandler struct {
	BlowerRPM  uint16 `json:"blowerRPM"`
	AirFlowCFM uint16 `json:"airFlowCFM"`
	ElecHeat   bool   `json:"elecHeat"`
}

type HeatPump struct {
	CoilTemp    float32 `json:"coilTemp"`
	OutsideTemp float32 `json:"outsideTemp"`
	Stage       uint8   `json:"stage"`
}

var infinity *InfinityProtocol

func holdTime(ht uint16) string {
	if ht == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d", ht/60, ht % 60)
}

func getZ0Config() (*TStatZone0Config, bool) {
	cfg := TStatZoneParams{}
	ok := infinity.ReadTable(devTSTAT, &cfg)
	if !ok {
		return nil, false
	}

	params := TStatCurrentParams{}
	ok = infinity.ReadTable(devTSTAT, &params)
	if !ok {
		return nil, false
	}

	holdZ1 := cfg.ZoneHold&0x01 != 0
	holdZ2 := cfg.ZoneHold&0x02 != 0
	holdZ3 := cfg.ZoneHold&0x04 != 0
	holdZ4 := cfg.ZoneHold&0x08 != 0

	return &TStatZone0Config{
		CurrentTempZ1:     params.Z1CurrentTemp,
		CurrentTempZ2:     params.Z2CurrentTemp,
		CurrentTempZ3:     params.Z3CurrentTemp,
		CurrentTempZ4:     params.Z4CurrentTemp,
		CurrentHumidityZ1: params.Z1CurrentHumidity,
		CurrentHumidityZ2: params.Z2CurrentHumidity,
		CurrentHumidityZ3: params.Z3CurrentHumidity,
		CurrentHumidityZ4: params.Z4CurrentHumidity,
		OutdoorTemp:       params.OutdoorAirTemp,
		Mode:              rawModeToString(params.Mode & 0xf),
		Stage:             params.Mode >> 5,
		FanModeZ1:         rawFanModeToString(cfg.Z1FanMode),
		FanModeZ2:         rawFanModeToString(cfg.Z2FanMode),
		FanModeZ3:         rawFanModeToString(cfg.Z3FanMode),
		FanModeZ4:         rawFanModeToString(cfg.Z4FanMode),
		HoldZ1:            &holdZ1,
		HoldZ2:            &holdZ2,
		HoldZ3:            &holdZ3,
		HoldZ4:            &holdZ4,
		HeatSetpointZ1:    cfg.Z1HeatSetpoint,
		CoolSetpointZ1:    cfg.Z1CoolSetpoint,
		HeatSetpointZ2:    cfg.Z2HeatSetpoint,
		CoolSetpointZ2:    cfg.Z2CoolSetpoint,
		HeatSetpointZ3:    cfg.Z3HeatSetpoint,
		CoolSetpointZ3:    cfg.Z3CoolSetpoint,
		HeatSetpointZ4:    cfg.Z4HeatSetpoint,
		CoolSetpointZ4:    cfg.Z4CoolSetpoint,
		HoldDurationZ1:    holdTime(cfg.Z1HoldDuration),
		HoldDurationZ2:    holdTime(cfg.Z2HoldDuration),
		HoldDurationZ3:    holdTime(cfg.Z3HoldDuration),
		HoldDurationZ4:    holdTime(cfg.Z4HoldDuration),
		ZoneNameZ1:        string(cfg.Z1Name[:]),
		ZoneNameZ2:        string(cfg.Z2Name[:]),
		ZoneNameZ3:        string(cfg.Z3Name[:]),
		ZoneNameZ4:        string(cfg.Z4Name[:]),
		RawMode:           params.Mode,
	}, true
}

func getZ1Config() (*TStatZoneConfig, bool) {
	cfg := TStatZoneParams{}
	ok := infinity.ReadTable(devTSTAT, &cfg)
	if !ok {
		return nil, false
	}

	params := TStatCurrentParams{}
	ok = infinity.ReadTable(devTSTAT, &params)
	if !ok {
		return nil, false
	}

	hold := new(bool)
	*hold = cfg.ZoneHold&0x01 == 1

	return &TStatZoneConfig{
		CurrentTemp:     params.Z1CurrentTemp,
		CurrentHumidity: params.Z1CurrentHumidity,
		OutdoorTemp:     params.OutdoorAirTemp,
		Mode:            rawModeToString(params.Mode & 0xf),
		Stage:           params.Mode >> 5,
		FanMode:         rawFanModeToString(cfg.Z1FanMode),
		Hold:            hold,
		HeatSetpoint:    cfg.Z1HeatSetpoint,
		CoolSetpoint:    cfg.Z1CoolSetpoint,
		HoldDuration:    holdTime(cfg.Z1HoldDuration),
		HoldDurationMins: cfg.Z1HoldDuration,
		ZoneName:        string(cfg.Z1Name[:]),
		TargetHumidity:  cfg.Z1TargetHumidity,
		RawMode:         params.Mode,
	}, true
}

func getZ2Config() (*TStatZoneConfig, bool) {
	cfg := TStatZoneParams{}
	ok := infinity.ReadTable(devTSTAT, &cfg)
	if !ok {
		return nil, false
	}

	params := TStatCurrentParams{}
	ok = infinity.ReadTable(devTSTAT, &params)
	if !ok {
		return nil, false
	}

	hold := new(bool)
	*hold = cfg.ZoneHold&0x01 == 1

	return &TStatZoneConfig{
		CurrentTemp:     params.Z2CurrentTemp,
		CurrentHumidity: params.Z2CurrentHumidity,
		OutdoorTemp:     params.OutdoorAirTemp,
		Mode:            rawModeToString(params.Mode & 0xf),
		Stage:           params.Mode >> 5,
		FanMode:         rawFanModeToString(cfg.Z2FanMode),
		Hold:            hold,
		HeatSetpoint:    cfg.Z2HeatSetpoint,
		CoolSetpoint:    cfg.Z2CoolSetpoint,
		HoldDuration:    holdTime(cfg.Z2HoldDuration),
		ZoneName:        string(cfg.Z2Name[:]),
		RawMode:         params.Mode,
	}, true
}

func getZ3Config() (*TStatZoneConfig, bool) {
	cfg := TStatZoneParams{}
	ok := infinity.ReadTable(devTSTAT, &cfg)
	if !ok {
		return nil, false
	}

	params := TStatCurrentParams{}
	ok = infinity.ReadTable(devTSTAT, &params)
	if !ok {
		return nil, false
	}

	hold := new(bool)
	*hold = cfg.ZoneHold&0x01 == 1

	return &TStatZoneConfig{
		CurrentTemp:     params.Z3CurrentTemp,
		CurrentHumidity: params.Z3CurrentHumidity,
		OutdoorTemp:     params.OutdoorAirTemp,
		Mode:            rawModeToString(params.Mode & 0xf),
		Stage:           params.Mode >> 5,
		FanMode:         rawFanModeToString(cfg.Z3FanMode),
		Hold:            hold,
		HeatSetpoint:    cfg.Z3HeatSetpoint,
		CoolSetpoint:    cfg.Z3CoolSetpoint,
		RawMode:         params.Mode,
	}, true
}

func getZ4Config() (*TStatZoneConfig, bool) {
	cfg := TStatZoneParams{}
	ok := infinity.ReadTable(devTSTAT, &cfg)
	if !ok {
		return nil, false
	}

	params := TStatCurrentParams{}
	ok = infinity.ReadTable(devTSTAT, &params)
	if !ok {
		return nil, false
	}

	hold := new(bool)
	*hold = cfg.ZoneHold&0x01 == 1

	return &TStatZoneConfig{
		CurrentTemp:     params.Z4CurrentTemp,
		CurrentHumidity: params.Z4CurrentHumidity,
		OutdoorTemp:     params.OutdoorAirTemp,
		Mode:            rawModeToString(params.Mode & 0xf),
		Stage:           params.Mode >> 5,
		FanMode:         rawFanModeToString(cfg.Z4FanMode),
		Hold:            hold,
		HeatSetpoint:    cfg.Z4HeatSetpoint,
		CoolSetpoint:    cfg.Z4CoolSetpoint,
		RawMode:         params.Mode,
	}, true
}

func getTstatSettings() (*TStatSettings, bool) {
	tss := TStatSettings{}
	ok := infinity.ReadTable(devTSTAT, &tss)
	if !ok {
		return nil, false
	}

	return &TStatSettings{
		BacklightSetting: tss.BacklightSetting,
		AutoMode:         tss.AutoMode,
		DeadBand:         tss.DeadBand,
		CyclesPerHour:    tss.CyclesPerHour,
		SchedulePeriods:  tss.SchedulePeriods,
		ProgramsEnabled:  tss.ProgramsEnabled,
		TempUnits:        tss.TempUnits,
		DealerName:       tss.DealerName,
		DealerPhone:      tss.DealerPhone,
	}, true
}

func getAirHandler() (AirHandler, bool) {
	b := cache.get("blower")
	tb, ok := b.(*AirHandler)
	if !ok {
		return AirHandler{}, false
	}
	return *tb, true
}

func getHeatPump() (HeatPump, bool) {
	h := cache.get("heatpump")
	th, ok := h.(*HeatPump)
	if !ok {
		return HeatPump{}, false
	}
	return *th, true
}

func statePoller() {
	for {
		// called once for all zones
		c1, ok := getZ0Config()
		if ok {
			cache.update("tstat", c1)
		}

		time.Sleep(time.Second * 1)
	}
}

func attachSnoops() {
	// Snoop Heat Pump responses
	infinity.snoopResponse(0x5000, 0x51ff, func(frame *InfinityFrame) {
		data := frame.data[3:]
		heatPump, ok := getHeatPump()
		if ok {
			if bytes.Equal(frame.data[0:3], []byte{0x00, 0x3e, 0x01}) {
				heatPump.CoilTemp = float32(binary.BigEndian.Uint16(data[2:4])) / float32(16)
				heatPump.OutsideTemp = float32(binary.BigEndian.Uint16(data[0:2])) / float32(16)
				log.Debugf("heat pump coil temp is: %f", heatPump.CoilTemp)
				log.Debugf("heat pump outside temp is: %f", heatPump.OutsideTemp)
				cache.update("heatpump", &heatPump)
			} else if bytes.Equal(frame.data[0:3], []byte{0x00, 0x3e, 0x02}) {
				heatPump.Stage = data[0] >> 1
				log.Debugf("HP stage is: %d", heatPump.Stage)
				cache.update("heatpump", &heatPump)
			}
		}
	})

	// Snoop Air Handler responses
	infinity.snoopResponse(0x4000, 0x42ff, func(frame *InfinityFrame) {
		data := frame.data[3:]
		airHandler, ok := getAirHandler()
		if ok {
			if bytes.Equal(frame.data[0:3], []byte{0x00, 0x03, 0x06}) {
				airHandler.BlowerRPM = binary.BigEndian.Uint16(data[1:5])
				log.Debugf("blower RPM is: %d", airHandler.BlowerRPM)
				cache.update("blower", &airHandler)
			} else if bytes.Equal(frame.data[0:3], []byte{0x00, 0x03, 0x16}) {
				airHandler.AirFlowCFM = binary.BigEndian.Uint16(data[4:8])
				airHandler.ElecHeat = data[0]&0x03 != 0
				log.Debugf("air flow CFM is: %d", airHandler.AirFlowCFM)
				cache.update("blower", &airHandler)
			}
		}
	})
}

func main() {
	httpPort := flag.Int("httpport", 8080, "HTTP port to listen on")
	serialPort := flag.String("serial", "", "path to serial port")

	flag.Parse()

	if len(*serialPort) == 0 {
		fmt.Print("must provide serial\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.SetLevel(log.DebugLevel)

	infinity = &InfinityProtocol{device: *serialPort}
	airHandler := new(AirHandler)
	heatPump := new(HeatPump)
	cache.update("blower", airHandler)
	cache.update("heatpump", heatPump)
	attachSnoops()
	err := infinity.Open()
	if err != nil {
		log.Panicf("error opening serial port: %s", err.Error())
	}

	go statePoller()
	webserver(*httpPort)
}
