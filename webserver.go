package main

import (
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"golang.org/x/net/websocket"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func handleErrors(c *gin.Context) {
	c.Next()

	if len(c.Errors) > 0 {
		c.JSON(-1, c.Errors) // -1 == not override the current error code
	}
}

func webserver(port int) {
	r := gin.Default()
	r.Use(handleErrors) // attach error handling middleware

	api := r.Group("/api")

	api.GET("/tstat/settings", func(c *gin.Context) {
		tss, ok := getTstatSettings()
		if ok {
			c.JSON(200, tss)
		}
	})

	api.GET("/zone/0/config", func(c *gin.Context) {
		cfgZ0, ok := getZ0Config()
		if ok {
			c.JSON(200, cfgZ0)
		}
	})

	api.GET("/zone/1/config", func(c *gin.Context) {
		cfgZ1, ok := getZ1Config()
		if ok {
			c.JSON(200, cfgZ1)
		}
	})

	api.GET("/zone/2/config", func(c *gin.Context) {
		cfgZ2, ok := getZ2Config()
		if ok {
			c.JSON(200, cfgZ2)
		}
	})

	api.GET("/zone/3/config", func(c *gin.Context) {
		cfgZ3, ok := getZ3Config()
		if ok {
			c.JSON(200, cfgZ3)
		}
	})

	api.GET("/zone/4/config", func(c *gin.Context) {
		cfgZ4, ok := getZ4Config()
		if ok {
			c.JSON(200, cfgZ4)
		}
	})

	api.GET("/zone/1/airhandler", func(c *gin.Context) {
		ah, ok := getAirHandler()
		if ok {
			c.JSON(200, ah)
		}
	})

	api.GET("/zone/1/heatpump", func(c *gin.Context) {
		hp, ok := getHeatPump()
		if ok {
			c.JSON(200, hp)
		}
	})

	api.GET("/zone/1/vacation", func(c *gin.Context) {
		vac := TStatVacationParams{}
		ok := infinity.ReadTable(devTSTAT, &vac)
		if ok {
			c.JSON(200, vac.toAPI())
		}
	})

	api.PUT("/zone/1/vacation", func(c *gin.Context) {
		var args APIVacationConfig

		if c.Bind(&args) != nil {
			log.Printf("bind failed")
			return
		}

		params := TStatVacationParams{}
		flags := params.fromAPI(&args)

		infinity.WriteTable(devTSTAT, params, flags)

	})

	api.PUT("/zone/1/config", func(c *gin.Context) {
		var args TStatZone0Config

		if c.Bind(&args) == nil {
			params := TStatZoneParams{}
			flags := byte(0)

			if len(args.FanModeZ1) > 0 {
				mode, _ := stringFanModeToRaw(args.FanModeZ1)
				// FIXME: check for ok here
				params.Z1FanMode = mode
				flags |= 0x01
			}

			if args.Hold != nil {
				if *args.Hold {
					params.ZoneHold = 0x01
				} else {
					params.ZoneHold = 0x00
				}
				flags |= 0x02
			}

			if args.HeatSetpointZ1 > 0 {
				params.Z1HeatSetpoint = args.HeatSetpointZ1
				flags |= 0x04
			}

			if args.CoolSetpointZ1 > 0 {
				params.Z1CoolSetpoint = args.CoolSetpointZ1
				flags |= 0x08
			}

			if flags != 0 {
				log.Printf("calling doWrite with flags: %x", flags)
				infinity.WriteTable(devTSTAT, params, flags)
			}

			if len(args.Mode) > 0 {
				p := TStatCurrentParams{Mode: stringModeToRaw(args.Mode)}
				infinity.WriteTable(devTSTAT, p, 0x10)
			}
		} else {
			log.Printf("bind failed")
		}
	})

	api.PUT("/zone/2/config", func(c *gin.Context) {
		var args TStatZone0Config

		if c.Bind(&args) == nil {
			params := TStatZoneParams{}
			flags := byte(0)

			if len(args.FanModeZ2) > 0 {
				mode, _ := stringFanModeToRaw(args.FanModeZ2)
				// FIXME: check for ok here
				params.Z2FanMode = mode
				flags |= 0x01
			}

			if args.Hold != nil {
				if *args.Hold {
					params.ZoneHold = 0x01
				} else {
					params.ZoneHold = 0x00
				}
				flags |= 0x02
			}

			if args.HeatSetpointZ2 > 0 {
				params.Z2HeatSetpoint = args.HeatSetpointZ2
				flags |= 0x04
			}

			if args.CoolSetpointZ2 > 0 {
				params.Z2CoolSetpoint = args.CoolSetpointZ2
				flags |= 0x08
			}

			if flags != 0 {
				log.Printf("calling doWrite with flags: %x", flags)
				infinity.WriteTable(devTSTAT, params, flags)
			}

			if len(args.Mode) > 0 {
				p := TStatCurrentParams{Mode: stringModeToRaw(args.Mode)}
				infinity.WriteTable(devTSTAT, p, 0x10)
			}
		} else {
			log.Printf("bind failed")
		}
	})

	api.PUT("/zone/3/config", func(c *gin.Context) {
		var args TStatZone0Config

		if c.Bind(&args) == nil {
			params := TStatZoneParams{}
			flags := byte(0)

			if len(args.FanModeZ3) > 0 {
				mode, _ := stringFanModeToRaw(args.FanModeZ3)
				// FIXME: check for ok here
				params.Z3FanMode = mode
				flags |= 0x01
			}

			if args.Hold != nil {
				if *args.Hold {
					params.ZoneHold = 0x01
				} else {
					params.ZoneHold = 0x00
				}
				flags |= 0x02
			}

			if args.HeatSetpointZ3 > 0 {
				params.Z3HeatSetpoint = args.HeatSetpointZ3
				flags |= 0x04
			}

			if args.CoolSetpointZ3 > 0 {
				params.Z3CoolSetpoint = args.CoolSetpointZ3
				flags |= 0x08
			}

			if flags != 0 {
				log.Printf("calling doWrite with flags: %x", flags)
				infinity.WriteTable(devTSTAT, params, flags)
			}

			if len(args.Mode) > 0 {
				p := TStatCurrentParams{Mode: stringModeToRaw(args.Mode)}
				infinity.WriteTable(devTSTAT, p, 0x10)
			}
		} else {
			log.Printf("bind failed")
		}
	})

	api.PUT("/zone/4/config", func(c *gin.Context) {
		var args TStatZone0Config

		if c.Bind(&args) == nil {
			params := TStatZoneParams{}
			flags := byte(0)

			if len(args.FanModeZ4) > 0 {
				mode, _ := stringFanModeToRaw(args.FanModeZ4)
				// FIXME: check for ok here
				params.Z4FanMode = mode
				flags |= 0x01
			}

			if args.Hold != nil {
				if *args.Hold {
					params.ZoneHold = 0x01
				} else {
					params.ZoneHold = 0x00
				}
				flags |= 0x02
			}

			if args.HeatSetpointZ4 > 0 {
				params.Z4HeatSetpoint = args.HeatSetpointZ4
				flags |= 0x04
			}

			if args.CoolSetpointZ4 > 0 {
				params.Z4CoolSetpoint = args.CoolSetpointZ4
				flags |= 0x08
			}

			if flags != 0 {
				log.Printf("calling doWrite with flags: %x", flags)
				infinity.WriteTable(devTSTAT, params, flags)
			}

			if len(args.Mode) > 0 {
				p := TStatCurrentParams{Mode: stringModeToRaw(args.Mode)}
				infinity.WriteTable(devTSTAT, p, 0x10)
			}
		} else {
			log.Printf("bind failed")
		}
	})

	api.GET("/raw/:device/:table", func(c *gin.Context) {
		matched, _ := regexp.MatchString("^[a-f0-9]{4}$", c.Param("device"))
		if !matched {
			c.AbortWithError(400, errors.New("name must be a 4 character hex string"))
			return
		}
		matched, _ = regexp.MatchString("^[a-f0-9]{6}$", c.Param("table"))
		if !matched {
			c.AbortWithError(400, errors.New("table must be a 6 character hex string"))
			return
		}

		d, _ := strconv.ParseUint(c.Param("device"), 16, 16)
		a, _ := hex.DecodeString(c.Param("table"))
		var addr InfinityTableAddr
		copy(addr[:], a[0:3])
		raw := InfinityProtocolRawRequest{&[]byte{}}

		success := infinity.Read(uint16(d), addr, raw)

		if success {
			c.JSON(200, gin.H{"response": hex.EncodeToString(*raw.data)})
		} else {
			c.AbortWithError(504, errors.New("timed out waiting for response"))
		}
	})

	api.GET("/ws", func(c *gin.Context) {
		h := websocket.Handler(attachListener)
		h.ServeHTTP(c.Writer, c.Request)
	})

	r.StaticFS("/ui", assetFS())
	// r.Static("/ui", "github.com/acd/infinitease/assets")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "ui")
	})

	r.Run(":" + strconv.Itoa(port)) // listen and server on 0.0.0.0:8080
}

func attachListener(ws *websocket.Conn) {
	listener := &EventListener{make(chan []byte, 32)}

	defer func() {
		Dispatcher.deregister <- listener
		log.Printf("closing websocket")
		err := ws.Close()
		if err != nil {
			log.Println("error on ws close:", err.Error())
		}
	}()

	Dispatcher.register <- listener

	// log.Printf("dumping cached data")
	for source, data := range cache {
		// log.Printf("dumping %s", source)
		ws.Write(serializeEvent(source, data))
	}

	// wait for events
	for {
		select {
		case message, ok := <-listener.ch:
			if !ok {
				log.Printf("read from listener.ch was not okay")
				return
			} else {
				_, err := ws.Write(message)
				if err != nil {
					log.Printf("error on websocket write: %s", err.Error())
					return
				}
			}
		}
	}
}
