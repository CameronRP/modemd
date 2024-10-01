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
	"time"

	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"
)

const (
	dbusName = "org.cacophony.modemd"
	dbusPath = "/org/cacophony/modemd"
)

type service struct {
	mc *ModemController
}

func sendModemConnectedSignal() error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	conn.Emit(dbusPath, dbusName+".ModemConnected", true)
	log.Println("Sent modem connected signal.")
	return nil
}

func startService(mc *ModemController) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	reply, err := conn.RequestName(dbusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return errors.New("names already taken")
	}

	s := &service{
		mc: mc,
	}
	conn.Export(s, dbusPath, dbusName)
	conn.Export(genIntrospectable(s), dbusPath, "org.freedesktop.DBus.Introspectable")
	return nil
}

func genIntrospectable(v interface{}) introspect.Introspectable {
	node := &introspect.Node{
		Interfaces: []introspect.Interface{{
			Name:    dbusName,
			Methods: introspect.Methods(v),
		}},
	}
	return introspect.NewIntrospectable(node)
}

// StayOn will keep the modem on for a set amount of time
func (s service) StayOn() *dbus.Error {
	s.mc.NewOnRequest()
	return nil
}

func (s service) StayOnFor(minutes int) *dbus.Error {
	err := s.mc.StayOnUntil(time.Now().Add(time.Duration(minutes) * time.Minute))
	if err != nil {
		return makeDbusError("StayOnFor", err)
	}
	return nil
}

func (s service) StayOffFor(minutes int) *dbus.Error {
	err := s.mc.StayOffUntil(time.Now().Add(time.Duration(minutes) * time.Minute))
	if err != nil {
		return makeDbusError("StayOffFor", err)
	}
	return nil
}

func (s service) GetStatus() (map[string]interface{}, *dbus.Error) {
	status, err := s.mc.GetStatus()
	if err != nil {
		log.Println(err)
		return nil, makeDbusError("GetStatus", err)
	}
	return status, nil
}

func (s service) SetAPN(apn string) *dbus.Error {
	log.Println("Setting APN to", apn)
	err := s.mc.setAPN(apn)
	if err != nil {
		log.Println(err)
		return makeDbusError("SetAPN", err)
	}
	return nil
}

func (s service) RunATCommand(atCommand string) (string, string, *dbus.Error) {
	if s.mc.Modem != nil && !s.mc.Modem.ATReady {
		return "", "", makeDbusError("RunATCommand", errors.New("modem not ready for AT commands"))
	}
	totalOut, out, err := s.mc.RunATCommandTotalOutput(atCommand)
	if err != nil {
		log.Println(err)
		return "", "", makeDbusError("RunATCommand", err)
	}
	return totalOut, out, nil
}

/*
func (s service) GPSOn() *dbus.Error {
	err := s.mc.EnableGPS()
	if err != nil {
		return makeDbusError("GPSOn", err)
	}
	return nil
}
*/
/*
func (s service) GPSOff() *dbus.Error {
	err := s.mc.DisableGPS()
	if err != nil {
		return makeDbusError("GPSOff", err)
	}
	return nil
}
*/

func makeDbusError(name string, err error) *dbus.Error {
	return &dbus.Error{
		Name: dbusName + name,
		Body: []interface{}{err.Error()},
	}
}
