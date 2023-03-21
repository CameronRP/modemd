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
	"time"

	"github.com/TheCacophonyProject/go-config"
)

type ModemdConfig struct {
	ModemsConfig           []config.Modem
	TestHosts              []string
	TestInterval           time.Duration
	PowerPin               string
	InitialOnDuration      time.Duration
	FindModemDuration      time.Duration
	ConnectionTimeout      time.Duration
	PingWaitTime           time.Duration
	PingRetries            int
	RequestOnDuration      time.Duration
	RetryInterval          time.Duration
	RetryFindModemInterval time.Duration
	MaxOffDuration         time.Duration
	MinConnDuration        time.Duration
}

func ParseModemdConfig(configDir string) (*ModemdConfig, error) {
	conf, err := config.New(configDir)
	if err != nil {
		return nil, err
	}

	mdConf := config.DefaultModemd()
	if err := conf.Unmarshal(config.ModemdKey, &mdConf); err != nil {
		return nil, err
	}

	testHosts := config.DefaultTestHosts()
	if err := conf.Unmarshal(config.TestHostsKey, &testHosts); err != nil {
		return nil, err
	}

	gpio := config.DefaultGPIO()
	if err := conf.Unmarshal(config.GPIOKey, &gpio); err != nil {
		return nil, err
	}

	return &ModemdConfig{
		ModemsConfig:           mdConf.Modems,
		TestHosts:              testHosts.URLs,
		TestInterval:           mdConf.TestInterval,
		PowerPin:               gpio.ModemPower,
		InitialOnDuration:      mdConf.InitialOnDuration,
		FindModemDuration:      mdConf.FindModemTimeout,
		ConnectionTimeout:      mdConf.ConnectionTimeout,
		PingWaitTime:           testHosts.PingWaitTime,
		PingRetries:            testHosts.PingRetries,
		RequestOnDuration:      mdConf.RequestOnDuration,
		RetryInterval:          mdConf.RetryInterval,
		RetryFindModemInterval: mdConf.RetryFindModemInterval,
		MinConnDuration:        mdConf.MinConnDuration,
		MaxOffDuration:         mdConf.MaxOffDuration,
	}, nil
}
