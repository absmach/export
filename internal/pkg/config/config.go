// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package writers contain the domain concept definitions needed to
// support Mainflux writer services functionality.
package export

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

type MQTTConf struct {
	Host        string `toml:"host" mapstructure:"host"`
	Username    string `toml:"username" mapstructure:"username"`
	Password    string `toml:"password" mapstructure:"password"`
	MTLS        bool   `toml:"mtls" mapstructure:"mtls"`
	SkipTLSVer  bool   `toml:"skip_tls_ver" mapstructure:"skip_tls_ver"`
	Retain      bool   `toml:"retain" mapstructure:"retain"`
	QoS         int    `toml:"qos" mapstructure:"qos"`
	Channel     string `toml:"channel" mapstructure:"channel"`
	CAPath      string `toml:"ca" mapstructure:"ca"`
	CertPath    string `toml:"cert" mapstructure:"cert"`
	PrivKeyPath string `toml:"priv_key" mapstructure:"priv_key"`
	CA          []byte
	Cert        tls.Certificate
}

type ServerConf struct {
	NatsURL  string `toml:"nats" mapstructure:"nats"`
	LogLevel string `toml:"log_level" mapstructure:"log_level"`
	Port     string `toml:"port" mapstructure:"port"`
}
type RoutesConf struct {
	Route []Route `toml:"routes" mapstructure:"routes"`
}
type Config struct {
	Server ServerConf   `toml:"exp" mapstructure:"exp"`
	Routes []RoutesConf `toml:"routes" mapstructure:"routes"`
	MQTT   MQTTConf     `toml:"mqtt" mapstructure:"mqtt"`
	File   string
}

type Route struct {
	MqttTopic *string `toml:"mqtt_topic" mapstructure:"mqtt_topic"`
	NatsTopic *string `toml:"nats_topic" mapstructure:"nats_topic"`
	SubTopic  *string `toml:"subtopic", mapstructure:"subtopic"`
	Type      *string `toml:"type", mapstructure:"type"`
}

func New(sc ServerConf, rc []RoutesConf, mc MQTTConf, file string) *Config {
	ac := Config{
		Server: sc,
		Routes: rc,
		MQTT:   mc,
		File:   file,
	}

	return &ac
}

// Save - store config in a file
func (c *Config) Save() error {
	b, err := toml.Marshal(*c)
	if err != nil {
		fmt.Printf("Error reading config file: %s", err)
		return err
	}

	if err := ioutil.WriteFile(c.File, b, 0644); err != nil {
		fmt.Printf("Error writing toml: %s", err)
		return err
	}

	return nil
}

// Read - retrieve config from a file
func (c *Config) Read() error {
	data, err := ioutil.ReadFile(c.File)
	if err != nil {
		fmt.Printf("Error reading config file: %s", err)
		return err
	}

	if err := toml.Unmarshal(data, c); err != nil {
		fmt.Printf("Error unmarshaling toml: %s", err)
		return err
	}

	return nil
}
