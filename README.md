# THIS FORK IS A WORK IN PROGRESS

This fork of infinitive has added read/write API and UI for multi-zone Infinity systems.  It currently (as of 8-Aug-2023) works and has
been tested on a 2-zone system but it should work up to 4 or 8 zones.  The UI adapts to show the zones that appear to be in use.

This code has been adapted for zoned systems from extensive previous work of others.  I (author of zoning enhancments) have not
yet been able to test everything and have received very little feedback so far.  Active development and testing is still under way.

In particular we still need to look into the following:
  * Not sure whether heating mode is reflected correctly in the UI or API.  Original work supported
    Heat Pump but we are now testing it on a system with AC and a gas heater.  Should have plenty of data in a
    few months.
  * Not sure how the updated UI will display the name of the one zone on a one-zone system - perhaps the zone name should be
    suppressed in that case if it is not commonly set up in a non-zone controller.
  * For bonus points, trying to figure out how Dehumidify mode is represented so we can reflect it in th UI/API
  * Will be adding reporting of automated zone damper status - mostly for fun I suppose
  * Fine-tune the detection of actual configured zones - currently using heuristic "currentTemp < 255" but hoping the acutal zone configs are hiding in there somewhere
  * Rebase to Will1604 fork or pick up backend comms changes and API enhancements
  * more updates to README
  * Working on a Home Assistant custom component for multi-zone use and which leverages the ws API used by the UI for faster
    (push) updates and less load on the serial bus.  The existing hass-infinitive component from @mww012
    works but only shows the 1st zone - I currently have a hacked-up version that supports multiple zones via inefficient polling.

This README has been updated with some info about this fork but more needs to be written.

# infinitive
Infinitive impersonates a SAM on a Carrier Infinity system management bus. 

This fork implements read/write and UI for multiple-zone systems.  It is of course backward compatible to a 1-zone system.

## **DISCLAIMER**
**The software and hardware described here interacts with a proprietary system using information gleaned by reverse engineering.  Although it works fine for me, no guarantee or warranty is provided.  Use at your own risk to your HVAC system and yourself.**

## Getting started
#### Hardware setup
I've done all my development on a Raspberry Pi, although any reasonably performant Linux system with an RS-485 interface should work.  I chose the Pi 3 since the built-in WiFi saved me the hassle of running Ethernet to my furnace.  I'm not sure if older Pis have enough horsepower to run the Infinitive software.  If you give it a shot and are successful, please let me know so I can update this information.

In addition to a Linux system, you'll need to source an adapter to communicate on the RS-485 bus.  I am using a FTDI chipset USB to RS-485 adapter that I bought from Amazon.  There are a variety of adapters on Amazon and eBay, although it may take a few attempts to find one that works reliably.

Once you have a RS-485 adapter you'll need to connect it to your ABCD bus. The easiest way to do this is by attaching new wires to the A and B terminals of the ABCD bus connector inside your furnace and connecting them to your adapter. The A and B lines are used for RS-485 communication, while C and D are 24V AC power. **Do not connect your RS-485 adapter to the C and D terminals unless you want to see its magic smoke.** 

#### Software
Download the Infinitive release appropriate for your architecture.

   * amd64:
```
$ wget -O infinitive https://github.com/acd/infinitive/releases/download/v0.2/infinitive.amd64
```
   * arm:
```
$ wget -O infinitive https://github.com/acd/infinitive/releases/download/v0.2/infinitive.arm
```

Start Infinitive, providing the HTTP port to listen on for the management interface and the path to the correct serial device.

```
$ ./infinitive -httpport=8080 -serial=/dev/ttyUSB0 
```

Logs are written to stderr.  For now I've been running Infinitive under screen.  If folks are interested in a proper start/stop script and log management, submit a pull request or let me know.

If the RS-485 adapter is properly connected to your ABCD bus you should immediately see Infinitive logging messages indicating it is receiving data, such as:

```
INFO[0000] read frame: 2001 -> 4001: READ     000302    
INFO[0000] read frame: 4001 -> 2001: RESPONSE 000302041100000414000004020000 
INFO[0000] read frame: 2001 -> 4001: READ     000316    
INFO[0000] read frame: 4001 -> 2001: RESPONSE 0003160000000003ba004a2f780100037a 
```

Browse to your host system's IP, with the port you provided on the command line, and you should see a page that looks similar to the following:

<img src="https://raw.githubusercontent.com/lurgh/infinitive/master/screenshot.png"/>

**Note:** I am not much of a frontend web developer.  I'd love to see pull requests for enhancements to the web interface.

There is a brief delay between altering a setting and Infinitive updating the information displayed.  This is due to Infinitive polling the thermostat settings once per second.

## Building from source

If you'd like to build Infinitive from source, first confirm you have a working Go environment (I've been using release 1.7.1).  Ensure your GOPATH and GOHOME are set correctly, then:

```
$ go get github.com/acd/infinitive
$ go build github.com/acd/infinitive
```
Note: If you make changes to the code or other resources in the assets directory you will need to rebuild the bindata_assetfs.go file. You will need the go-bindata-assetfs utility.
 
1. Install go-bindata-assetfs into your go folders

Details, and installation instructions are available here.

https://github.com/elazarl/go-bindata-assetfs

but can be summarized as:

```
$ go install github.com/go-bindata/go-bindata/...
$ go install github.com/elazarl/go-bindata-assetfs/...
```

2. Rebuild bindata_assetfs.go

From within the infinitive folder execute
```
$ go-bindata-assetfs assets/...
```


## JSON API

Infinitive exposes a JSON API to retrieve and manipulate thermostat parameters.

#### GET /api/zone/[Z]/config

Replace [Z] with any zone number 1-8.  If you want data for multiple zones, it's more efficient to use "GET /api/zones/config" to get all at once.


```json
{
   "currentTemp": 70,
   "currentHumidity": 50,
   "outdoorTemp": 50,
   "mode": "heat",
   "stage":2,
   "fanMode": "auto",
   "hold": true,
   "targetHumidity": 52,
   "zoneName": "Downstairs",
   "holdDuration": "1:50",
   "holdDurationMins": 110,
   "heatSetpoint": 68,
   "coolSetpoint": 74,
   "rawMode": 64
}
```
rawMode included for debugging purposes. It encodes stage and mode. 

Note that paramers stage, mode, outdoorTemp, and rawMode are global across all zones but for historical reasons they are present
in the per-zone query.

#### PUT /api/zone/[Z]/config

Replace [Z] with any zone number 1-8.  One or more parameters to write should be included in the JSON body.  Parameters that are not
mentioned are not changed.  The only parameters that are settable are "fanMode", "heatSetpoint", "coolSetpoint", and "hold", as well as the global
parameter "mode".

```json
{
   "mode": "auto",
   "fanMode": "auto",
   "hold": true,
   "heatSetpoint": 68,
   "coolSetpoint": 74
}
```

Valid write values for `mode` are `off`, `auto`, `heat`, and `cool`.
Additional read values for mode are `electric` and `heatpump` indicating "heat pump only" or "electric heat only" have been selected at the thermostat 
Values for `fanMode` are `auto`, `low`, `med`, and `high`.

#### GET /api/zones/config

This retrieves and returns data for all zones at once in a single JSON structure.  It's more efficient to use this
when you need multiple zones' data.  However, PUT is not supported with this structure currently so any PUTs must be done
using the per-zone API.

Note that this includes an array of the per-zone structures which includes the zone number in each one - that is, the zone number
is not necessarily related directly to the index in this array.  Also note that the "global" parameters (outdoorTemp, mode, stage, rawMode)
are properly at the top level of this
dictionary.  These parameters may for now also be inside the per-zone structures but should not be used and will eventually be removed.

```json
{
   "zones": [
      {
         "zoneNumber":1,
	 "currentTemp":79,
	 "currentHumidity":46,
	 "targetHumidity":0,
	 "zoneName":"Downstairs",
	 "fanMode":"low",
	 "hold":true,
	 "heatSetpoint":72,
	 "coolSetpoint":82,
	 "holdDuration":"",
	 "holdDurationMins":0,
	 "outdoorTemp":0,
	 "mode":"",
	 "stage":0,
	 "rawMode":0
      },
      {
         "zoneNumber":2,
	 "currentTemp":82,
	 "currentHumidity":46,
	 "targetHumidity":0,
	 "zoneName":"Upstairs",
	 "fanMode":"med",
	 "hold":false,
	 "heatSetpoint":73,
	 "coolSetpoint":83,
	 "holdDuration":"1:25",
	 "holdDurationMins":85,
	 "outdoorTemp":0,
	 "mode":"",
	 "stage":0,
	 "rawMode":0
      }
   ],
   "outdoorTemp":74,
   "mode":"cool",
   "stage":0,
   "rawMode":1
}

```
rawMode included for debugging purposes. It encodes stage and mode. 

#### GET /api/airhandler

This call is also supported as "GET /api/zone/1/airhandler" for backward compatibility but this is not per-zone data.

```json
{
	"blowerRPM":0,
	"airFlowCFM":0,
	"elecHeat":false
}
```

#### GET /api/heatpump

This call is also supported as "GET /api/zone/1/heatpump" for backward compatibility but this is not per-zone data.

```json
{
	"coilTemp":28.8125,
	"outsideTemp":31.375,
	"stage":2
}
```


#### GET /api/zone/1/vacation

```
{
   "active":false,
   "days":0,
   "minTemperature":56,
   "maxTemperature":84,
   "minHumidity":15,
   "maxHumidity":60,
   "fanMode":"auto"
}
```

#### PUT /api/zone/1/vacation

```
{
   "days":0,
   "minTemperature":56,
   "maxTemperature":84,
   "minHumidity":15,
   "maxHumidity":60,
   "fanMode":"auto"
}
```

All parameters are optional.  A single parameter may be updated by sending a JSON document containing only that parameter.  Vacation mode is disabled by setting `days` to `0`.  Valid values for `fanMode` are `auto`, `low`, `med`, and `high`.

## Details
#### ABCD bus
Infinity systems use a proprietary binary protocol for data exchange between system components.  These message are sent across an RS-485 serial bus which Carrier refers to as the ABCD bus.  Most systems usually includes an air-conditioning unit or heat pump, furnace, and thermostat.  The thermostat is responsible for enumerating other components of the system and managing their operation. 

The protocol has been reverse engineered as Carrier has not published a protocol specification.  The following resources provided invaluable assistance with my reverse engineering efforts:

* [Cocoontech's Carrier Infinity Thread](http://cocoontech.com/forums/topic/11372-carrier-infinity/)
* [Infinitude Wiki](https://github.com/nebulous/infinitude/wiki/Infinity-Protocol-Main)

Infinitive reads and writes information from the Infinity thermostat.  It also gathers data by passively observing traffic exchanged between the thermostat and other system components.

#### Bryant Evolution
I believe Infinitive should work with Bryant Evolution systems as they use the same ABCD bus.  Please let me know if you have success using Infinitive on a Bryant system.

#### Notes About Multi-Zone Systems

Multi-zone systems are supported in this version.  We have tested it on a 2-zone system but it should work at least up to 4 and most likely up to the
apparent 8-zone limit of the Infinity architecture.  Please get in touch if you have success or difficulty with a zoned system.

The UI will automatically show all the zones, listed in order of their index number.  The API can access a single zone's data at a tine, or all zones
in one go; if your application wants all the zone data then it's more efficient to use the latter since the per-zone APIs will be slower owing to each
one needing make redundant requests to the system.  The all-zones API is read-only; use the per-zone PUT method to make changes to a zone's configuration.

#### Unimplemented features

I don't use the thermostat's scheduling capabilities or vacation mode so Infinitive does not support them.  The schema and encoding of the scheduling data are fairly obvious so API support could be added if there is interest.  Reach out if this is something you'd like to see.  

#### Issues
##### rPi USB stack
The USB to RS-485 adapter I'm using periodically locks up due to what appear to be USB stack issues on the Raspberry Pi 3.  When this happens, reads on the serial file descriptor block forever and the kernel logs the following:
```
[491862.396039] ftdi_sio ttyUSB0: usb_serial_generic_read_bulk_callback - urb stopped: -32
```
Infinitive reopens the serial interface when it hasn't received any data in 5 seconds to workaround the issue.  Alternatively, forcing the Pi USB stack to USB 1.1 mode resolves the issue.  If you want to go this route, add `dwc_otg.speed=1` to `/boot/config.txt` and reboot the Pi.

##### Bogus data (fixed)
There was a long-standing problem wherein occasionally Infinitive's UI would display incorrect data via the web interface for a second.  This was due to a bug in the go code and has been fixed in this fork.  Leaving this note so others familiar with the README will see it.

#### See Also
[Infinitude](https://github.com/nebulous/infinitude) is another solution for managing Carrier HVAC systems.  It impersonates Carrier web services and provides an alternate interface for controlling Carrier Internet-enabled touchscreen thermostats.  It also supports passive snooping of the RS-485 bus and can decode and display some of the data.

#### Contact
Andrew Danforth (<adanforth@gmail.com>)
