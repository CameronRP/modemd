package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	goconfig "github.com/TheCacophonyProject/go-config"
)

type Modem struct {
	Name          string
	Netdev        string
	VendorProduct string
	ATReady       bool
	SimReady      bool
}

// NewModem return a new modem from the config
func NewModem(config goconfig.Modem) *Modem {
	m := &Modem{
		Name:          config.Name,
		Netdev:        config.NetDev,
		VendorProduct: config.VendorProductID,
	}
	return m
}

// PingTest will try connecting to one of the provides hosts
func (m *Modem) PingTest(timeoutSec int, retries int, hosts []string) bool {
	for i := retries; i > 0; i-- {
		for _, host := range hosts {
			cmd := exec.Command(
				"ping",
				"-I",
				m.Netdev,
				"-n",
				"-q",
				"-c1",
				fmt.Sprintf("-w%d", timeoutSec),
				host)
			if err := cmd.Run(); err == nil {
				return true
			}
		}
		if i > 1 {
			log.Printf("ping test failed. %d more retries\n", i-1)
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// IsDefaultRoute will check if the USB modem is connected
func (m *Modem) IsDefaultRoute() (bool, error) {
	outByte, err := exec.Command("ip", "route").Output()
	if err != nil {
		return false, err
	}
	out := string(outByte)
	lines := strings.Split(out, "\n")
	search := fmt.Sprintf(" dev %s ", m.Netdev)
	for _, line := range lines {
		if strings.HasPrefix(line, "default") && strings.Contains(line, search) {
			return true, nil
		}
	}
	return false, nil
}

/*
func (m *Modem) WaitForConnection(timeout int) (bool, error) {
	start := time.Now()
	for {
		def, err := m.IsDefaultRoute()
		if err != nil {
			return false, err
		}
		if def {
			return true, nil
		}
		if time.Since(start) > time.Second*time.Duration(timeout) {
			return false, nil
		}
		time.Sleep(time.Second)
	}
}
*/
