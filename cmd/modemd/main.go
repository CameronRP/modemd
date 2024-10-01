/*
modemd - Communicates with USB modems
Copyright (C) 2019, The Cacophony Project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"errors"
	"strconv"
	"time"

	"github.com/TheCacophonyProject/event-reporter/v3/eventclient"
	"github.com/TheCacophonyProject/go-config"
	"github.com/TheCacophonyProject/go-utils/logging"
	arg "github.com/alexflint/go-arg"
	"periph.io/x/periph/host"
)

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err)
	}
}

type Args struct {
	ConfigDir    string `arg:"-c,--config" help:"path to configuration directory"`
	Timestamps   bool   `arg:"-t,--timestamps" help:"include timestamps in log output"`
	RestartModem bool   `arg:"-r,--restart" help:"cycle the power to the USB port"`
	logging.LogArgs
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	args := Args{
		ConfigDir: config.DefaultConfigDir,
	}
	arg.MustParse(&args)
	return args
}

var version = "<not set>"
var log = logging.NewLogger("info")

func runMain() error {
	args := procArgs()

	log = logging.NewLogger(args.LogLevel)

	log.Printf("Running version: %s", version)

	if _, err := host.Init(); err != nil {
		return err
	}

	conf, err := ParseModemdConfig(args.ConfigDir)
	if err != nil {
		return err
	}

	log.Printf("%+v\n", conf)

	m := []config.Modem{
		{Name: "Qualcomm", NetDev: "usb0", VendorProductID: "1e0e:9011"},
	}

	mc := ModemController{
		StartTime: time.Now(),
		//ModemsConfig:           conf.ModemsConfig,
		ModemsConfig:           m,
		TestHosts:              conf.TestHosts,
		TestInterval:           conf.TestInterval,
		PowerPin:               conf.PowerPin,
		InitialOnDuration:      conf.InitialOnDuration,
		FindModemDuration:      conf.FindModemDuration,
		ConnectionTimeout:      conf.ConnectionTimeout,
		PingWaitTime:           conf.PingWaitTime,
		PingRetries:            conf.PingRetries,
		RequestOnDuration:      conf.RequestOnDuration,
		RetryInterval:          conf.RetryInterval,
		RetryFindModemInterval: conf.RetryFindModemInterval,
		MinConnDuration:        conf.MinConnDuration,
		MaxOffDuration:         conf.MaxOffDuration,
	}

	log.Println("Starting dbus service.")
	if err := startService(&mc); err != nil {
		return err
	}

	if !mc.ShouldBeOn() || args.RestartModem {
		if err := mc.SetModemPower(false); err != nil {
			return err
		}
	}

	for {
		// =========== Wait until modem should be on ===========
		if !mc.ShouldBeOn() {
			log.Println("Waiting until modem should be powered on.")
			for !mc.ShouldBeOn() {
				time.Sleep(5 * time.Second)
			}
		}
		if err := mc.SetModemPower(true); err != nil {
			return err
		}

		// =========== Finding modem ===========
		log.Println("Finding USB modem.")
		for retries := 3; retries > 0; retries-- {
			if mc.FindModem() {
				log.Printf("Found modem %s.\n", mc.Modem.Name)
				usbMode, err := mc.IsInUSBMode()
				if err != nil {
					return err
				}
				if !usbMode {
					if err := mc.EnableUSBMode(); err != nil {
						return err
					}
				}
				break
			} else {
				log.Printf("No USB modem found. Will cycle power %d more time(s) to find modem", retries)
			}
		}
		if mc.Modem == nil {
			log.Println("Failed to find modem.")
			log.Println("Making noModemFound event.")
			eventclient.AddEvent(eventclient.Event{
				Timestamp: time.Now(),
				Type:      "noModemFound",
			})
			mc.failedToFindModem = true
			continue
		}

		// ========== Checking for AT response from modem. =============
		log.Println("Waiting for AT command to respond.")
		for i := 0; i < 20; i++ {
			_, err := mc.RunATCommand("AT")
			if err == nil {
				mc.Modem.ATReady = true
				log.Println("AT command responding.")
				break
			}
			time.Sleep(3 * time.Second)
		}
		if !mc.Modem.ATReady {
			log.Println("Making noModemATCommandResponse event.")
			eventclient.AddEvent(eventclient.Event{
				Timestamp: time.Now(),
				Type:      "noModemATCommandResponse",
			})
			return errors.New("failed to get AT command response")
		}
		time.Sleep(5 * time.Second) // Wait a little bit longer or else might get AT ERRORS
		mc.Modem.ATReady = true
		if err := mc.DisableGPS(); err != nil {
			return err
		}

		// ========== Checking SIM card. =============
		log.Println("Checking SIM card.")
		for retries := 5; retries > 0; retries-- {
			simStatus, err := mc.CheckSimCard()
			if err == nil && simStatus == "READY" {
				mc.Modem.SimReady = true
				break
			}
			log.Printf("SIM card not ready. Will cycle power %d more time(s) to find SIM card", retries)
			time.Sleep(5 * time.Second)
		}
		if !mc.Modem.SimReady {
			mc.failedToFindSimCard = true
			makeModemEvent("noModemSimCard", &mc)
			continue
		}

		// ========== Checking signal strength. =============
		log.Println("Checking signal strength.")
		// TODO make configurable to how long it will try to find a connection
		gotSignal := false
		for i := 0; i < 5; i++ {
			strengthStr, _ := mc.signalStrength()
			strength, err := strconv.Atoi(strengthStr)
			if err == nil && strength != 99 {
				log.Printf("Signal strength: %s", strengthStr)
				gotSignal = true
				break
			}
			time.Sleep(3 * time.Second)
		}
		if !gotSignal {
			mc.lastFailedConnection = time.Now()
			makeModemEvent("noModemSignal", &mc)
			continue
		}

		// ========== Wait for connection to internet =============
		connected, err := mc.WaitForConnection()
		if err != nil {
			return err
		}
		if !connected {
			mc.lastFailedConnection = time.Now()
			makeModemEvent("modemPingFail", &mc)
			continue
		}

		connectionsFirstPing := true

		log.Println("Modem has connected to a network.")
		makeModemEvent("modemConnectedToNetwork", &mc)
		mc.connectedTime = time.Now()
		for {
			if mc.PingTest() {
				mc.lastSuccessfulPing = time.Now()
				if connectionsFirstPing {
					connectionsFirstPing = false
					sendModemConnectedSignal() // This allows programs to trigger events when the modem connects.
				}
			} else {
				log.Println("ping test failed")
				mc.lastFailedConnection = time.Now()
				break
			}
			if !mc.WaitForNextPingTest() {
				break
			}
		}

		log.Println("Powering off USB modem.")
		if err := mc.SetModemPower(false); err != nil {
			return err
		}
		mc.Modem = nil
	}
}

func makeModemEvent(eventType string, mc *ModemController) {
	log.Printf("Making modem event '%s'.", eventType)
	signalStrength, err := mc.signalStrength()
	if err != nil {
		log.Printf("Failed to get signal strength: %s", err)
	}
	band, err := mc.readBand()
	if err != nil {
		log.Printf("Failed to get band: %s", err)
	}
	simStatus, err := mc.CheckSimCard()
	if err != nil {
		log.Printf("Failed to get sim status: %s", err)
	}
	apn, err := mc.getAPN()
	if err != nil {
		log.Printf("Failed to get apn: %s", err)
	}
	simProvider, err := mc.readSimProvider()
	if err != nil {
		log.Printf("Failed to get sim provider: %s", err)
	}
	provider, accessTechnology, err := mc.readProvider()
	if err != nil {
		log.Printf("Failed to get provider: %s", err)
	}
	iccid, err := mc.readSimICCID()
	if err != nil {
		log.Printf("Failed to get iccid: %s", err)
	}

	eventclient.AddEvent(eventclient.Event{
		Timestamp: time.Now(),
		Type:      eventType,
		Details: map[string]interface{}{
			"signalStrength":   signalStrength,
			"band":             band,
			"simStatus":        simStatus,
			"apn":              apn,
			"provider":         provider,
			"accessTechnology": accessTechnology,
			"simProvider":      simProvider,
			"iccid":            iccid,
		},
	})
}
