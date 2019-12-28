// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package writers contain the domain concept definitions needed to
// support Mainflux writer services functionality.
package config

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
	CAPath      string `toml:"ca_path" mapstructure:"ca_path"`
	CertPath    string `toml:"cert_path" mapstructure:"cert_path"`
	PrivKeyPath string `toml:"priv_key_path" mapstructure:"priv_key_path"`
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
	Server ServerConf `toml:"exp" mapstructure:"exp"`
	Routes []Route    `toml:"routes" mapstructure:"routes"`
	MQTT   MQTTConf   `toml:"mqtt" mapstructure:"mqtt"`
	File   string
}

type Route struct {
	MqttTopic *string `toml:"mqtt_topic" mapstructure:"mqtt_topic"`
	NatsTopic *string `toml:"nats_topic" mapstructure:"nats_topic"`
	SubTopic  *string `toml:"subtopic", mapstructure:"subtopic"`
	Type      *string `toml:"type", mapstructure:"type"`
}

func New(sc ServerConf, rc []Route, mc MQTTConf, file string) *Config {
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
	file := c.File
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading config file: %s", err)
		return err
	}

	if err := toml.Unmarshal(data, c); err != nil {
		fmt.Printf("Error unmarshaling toml: %s", err)
		return err
	}
	c.File = file
	return nil
}

// ReadFromB - retrieve config from a byte
func (c *Config) ReadFromB(data []byte) error {
	if err := toml.Unmarshal(data, c); err != nil {
		fmt.Printf("Error unmarshaling toml: %s", err)
		return err
	}
	return nil
}
