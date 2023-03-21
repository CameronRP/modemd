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
	"log"
	"time"

	"github.com/TheCacophonyProject/event-reporter/v3/eventclient"
	"github.com/TheCacophonyProject/go-config"
	arg "github.com/alexflint/go-arg"
	"periph.io/x/periph/host"

	saltrequester "github.com/TheCacophonyProject/salt-updater"
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

func runMain() error {
	args := procArgs()
	if !args.Timestamps {
		log.SetFlags(0) // Removes default timestamp flag
	}
	log.Printf("running version: %s", version)

	log.Print("init gpio")
	if _, err := host.Init(); err != nil {
		return err
	}

	conf, err := ParseModemdConfig(args.ConfigDir)
	if err != nil {
		return err
	}

	log.Printf("%+v\n", conf)

	mc := ModemController{
		StartTime:              time.Now(),
		ModemsConfig:           conf.ModemsConfig,
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

	log.Println("starting dbus service")
	if err := startService(&mc); err != nil {
		return err
	}

	if !mc.ShouldBeOn() || args.RestartModem {
		log.Println("powering off USB modem")
		mc.SetModemPower(false)
	}

	for {
		log.Println("waiting until modem should be powered on")
		for !mc.ShouldBeOn() {
			time.Sleep(5 * time.Second)
		}

		log.Println("powering on USB modem")
		mc.SetModemPower(true)

		log.Println("finding USB modem")
		retries := 3
		for mc.ShouldBeOn() {
			if mc.FindModem() {
				log.Printf("found modem %s\n", mc.Modem.Name)
				break
			}
			retries--
			if retries < 1 {
				log.Println("failed to find USB modem, will check again later")
				mc.lastFailedFindModem = time.Now()
				mc.SetModemPower(false)
				break
			} else {
				log.Printf("no USB modem found. Will cycle power %d more time(s) to find modem", retries)
			}
			mc.CycleModemPower()
		}

		if mc.Modem != nil {
			log.Println("waiting for modem to connect to a network")
			connected, err := mc.WaitForConnection()
			if err != nil {
				return err
			}
			connectionsFirstPing := true
			if connected {
				log.Println("modem has connected to a network")
				mc.connectedTime = time.Now()
				for {
					if mc.PingTest() {
						mc.lastSuccessfulPing = time.Now()
						if connectionsFirstPing {
							connectionsFirstPing = false
							eventclient.UploadEvents() // Upload events each time connecting.
							saltrequester.RunPing()    // Ping salt server triggering scheduled commands to run.
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
			} else {
				log.Println("modem failed to connect to a network")
			}
		}

		mc.Modem = nil

		log.Println("powering off USB modem")
		mc.SetModemPower(false)
	}
}
