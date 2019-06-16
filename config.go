package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type sensorList []sensor

func (s *sensorList) String() string {
	if len(*s) == 0 {
		return ""
	}

	sensors := []string{}
	for _, sensor := range *s {
		sensors = append(sensors, sensor.String())
	}
	return fmt.Sprintf("%s", sensors)
}

func (s *sensorList) Type() string {
	return "address"
}

func (s *sensorList) Set(value string) error {
	sensor, err := parseSensor(value)
	if err != nil {
		return fmt.Errorf("can not parse sensor: %s", err)
	}

	*s = append(*s, sensor)
	return nil
}

type sensor struct {
	Name       string
	MacAddress string
}

func (s sensor) String() string {
	if s.Name == "" {
		return s.MacAddress
	}

	return fmt.Sprintf("%s (%s)", s.Name, s.MacAddress)
}

func parseSensor(value string) (sensor, error) {
	if len(value) == 0 {
		return sensor{}, errors.New("empty string")
	}

	tokens := strings.SplitN(value, "=", 2)
	if len(tokens) == 1 {
		return sensor{
			MacAddress: tokens[0],
		}, nil
	}

	return sensor{
		Name:       tokens[0],
		MacAddress: tokens[1],
	}, nil
}

type config struct {
	ListenAddr      string
	Sensors         sensorList
	Device          string
	RefreshDuration time.Duration
}

func parseConfig() (config, error) {
	result := config{
		ListenAddr:      ":9294",
		Device:          "hci0",
		RefreshDuration: 2 * time.Minute,
	}

	pflag.StringVarP(&result.ListenAddr, "addr", "a", result.ListenAddr, "Address to listen on for connections.")
	pflag.VarP(&result.Sensors, "sensor", "s", "MAC-address of sensor to collect data from. Can be specified multiple times.")
	pflag.StringVarP(&result.Device, "adapter", "i", result.Device, "Bluetooth device to use for communication.")
	pflag.DurationVarP(&result.RefreshDuration, "refresh-duration", "r", result.RefreshDuration, "Interval used for refreshing data from bluetooth devices.")
	pflag.Parse()

	if len(result.Sensors) == 0 {
		return result, errors.New("need to provide at least one sensor")
	}

	if len(result.Device) == 0 {
		return result, errors.New("need to provide a bluetooth device")
	}

	if result.RefreshDuration < time.Minute {
		log.Warnf("Refresh durations below one minute are discouraged: %s", result.RefreshDuration)
	}

	return result, nil
}
