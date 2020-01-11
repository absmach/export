// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/export/internal/app/export/api"
	"github.com/mainflux/export/pkg/config"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

const (
	svcName           = "export"
	defNatsURL        = nats.DefaultURL
	defLogLevel       = "debug"
	defPort           = "8170"
	defMqttHost       = "tcp://localhost:1883"
	defMqttUsername   = ""
	defMqttPassword   = ""
	defMqttChannel    = ""
	defMqttSkipTLSVer = "true"
	defMqttMTLS       = "false"
	defMqttCA         = "ca.crt"
	defMqttQoS        = "0"
	defMqttRetain     = false
	defMqttCert       = "thing.cert"
	defMqttPrivKey    = "thing.key"
	defConfigFile     = "../configs/config.toml"

	envNatsURL  = "MF_NATS_URL"
	envLogLevel = "MF_EXPORT_LOG_LEVEL"
	envPort     = "MF_EXPORT_PORT"

	envMqttHost       = "MF_EXPORT_MQTT_HOST"
	envMqttUsername   = "MF_EXPORT_MQTT_USERNAME"
	envMqttPassword   = "MF_EXPORT_MQTT_PASSWORD"
	envMqttChannel    = "MF_EXPORT_MQTT_CHANNEL"
	envMqttSkipTLSVer = "MF_EXPORT_MQTT_SKIP_TLS"
	envMqttMTLS       = "MF_EXPORT_MQTT_MTLS"
	envMqttCA         = "MF_EXPORT_MQTT_CA"
	envMqttQoS        = "MF_EXPORT_MQTT_QOS"
	envMqttRetain     = "MF_EXPORT_MQTT_RETAIN"
	envMqttCert       = "MF_EXPORT_MQTT_CLIENT_CERT"
	envMqttPrivKey    = "MF_EXPORT_MQTT_CLIENT_PK"
	envConfigFile     = "MF_EXPORT_CONF_PATH"

	keyNatsURL        = "exp.nats"
	keyExportPort     = "exp.port"
	keyExportLogLevel = "exp.log_level"
	keyMqttMTls       = "mqtt.mtls"
	keyMqttSkipTLS    = "mqtt.skip_tls_ver"
	keyMqttUrl        = "mqtt.url"
	keyMqttClientCert = "mqtt.cert"
	keyMqttPrivKey    = "mqtt.priv_key"
	keyMqttQOS        = "mqtt.qos"
	keyMqttRetain     = "mqtt.retain"
	keyMqttCA         = "mqtt.ca"
	keyMqttPassword   = "mqtt.password"
	keyMqttUsername   = "mqtt.username"
	keyMqttChannel    = "mqtt.channel"
	keyChanCfg        = "channels"
)

func main() {
	cfg, err := loadConfigs()
	if err != nil {
		log.Fatalf(err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.Server.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	nc, err := nats.Connect(cfg.Server.NatsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s %s", err, cfg.Server.NatsURL))
		os.Exit(1)
	}
	defer nc.Close()

	svc, err := export.New(nc, *cfg, logger)
	if err != nil {
		log.Fatalf(fmt.Sprintf("Failed to start service %s", err.Error()))
	}
	svc.Start(svcName)

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go startHTTPService(svc, cfg.Server.Port, logger, errs)

	err = <-errs
	logger.Error(fmt.Sprintf("export writer service terminated: %s", err))
}

func loadConfigs() (*config.Config, error) {
	configFile := mainflux.Env(envConfigFile, defConfigFile)
	sc := config.ServerConf{}
	rc := []config.Route{}
	mc := config.MQTTConf{}

	cfg := config.NewConfig(sc, rc, mc, configFile)
	err := cfg.ReadFile()
	if err != nil {
		mqttSkipTLSVer, err := strconv.ParseBool(mainflux.Env(envMqttSkipTLSVer, defMqttSkipTLSVer))
		if err != nil {
			mqttSkipTLSVer = false
		}
		mqttMTLS, err := strconv.ParseBool(mainflux.Env(envMqttMTLS, defMqttMTLS))
		if err != nil {
			mqttMTLS = false
		}
		mqttRetain, err := strconv.ParseBool(mainflux.Env(envMqttMTLS, defMqttMTLS))
		if err != nil {
			mqttRetain = false
		}

		q, err := strconv.ParseInt(mainflux.Env(envMqttQoS, defMqttQoS), 10, 64)
		if err != nil {
			q = 0
		}
		QoS := int(q)

		sc := config.ServerConf{
			NatsURL:  mainflux.Env(envNatsURL, defNatsURL),
			LogLevel: mainflux.Env(envLogLevel, defLogLevel),
			Port:     mainflux.Env(envPort, defPort),
		}

		mc := config.MQTTConf{
			Host:     mainflux.Env(envMqttHost, defMqttHost),
			Password: mainflux.Env(envMqttPassword, defMqttPassword),
			Username: mainflux.Env(envMqttUsername, defMqttUsername),

			Retain:     mqttRetain,
			QoS:        QoS,
			MTLS:       mqttMTLS,
			SkipTLSVer: mqttSkipTLSVer,

			CAPath:      mainflux.Env(envMqttCA, defMqttCA),
			CertPath:    mainflux.Env(envMqttCert, defMqttCert),
			PrivKeyPath: mainflux.Env(envMqttPrivKey, defMqttPrivKey),
		}
		mqttTopic := mainflux.Env(envMqttChannel, defMqttChannel)
		natsTopic := "*"
		rc := []config.Route{{
			MqttTopic: mqttTopic,
			NatsTopic: natsTopic,
		}}
		cfg := config.NewConfig(sc, rc, mc, configFile)
		err = loadCertificate(cfg)
		if err != nil {
			return cfg, err
		}
		err = cfg.Save()
		if err != nil {
			log.Println(fmt.Sprintf("Failed to save %s", err))
		}
		log.Println(fmt.Sprintf("Configuration loaded from environment, initial %s saved", configFile))
		return cfg, nil
	}
	err = loadCertificate(cfg)
	if err != nil {
		return cfg, err
	}
	log.Println(fmt.Sprintf("Configuration loaded from file %s", configFile))
	return cfg, nil
}

func loadCertificate(cfg *config.Config) errors.Error {

	caByte := []byte{}
	cert := tls.Certificate{}
	if cfg.MQTT.MTLS {
		caFile, err := os.Open(cfg.MQTT.CAPath)
		defer caFile.Close()
		if err != nil {
			return errors.Wrap(nil, err)
		}
		caByte, _ = ioutil.ReadAll(caFile)

		clientCert, err := os.Open(cfg.MQTT.CertPath)
		defer clientCert.Close()
		if err != nil {
			return errors.Wrap(nil, err)
		}
		cc, _ := ioutil.ReadAll(clientCert)

		privKey, err := os.Open(cfg.MQTT.PrivKeyPath)
		defer clientCert.Close()
		if err != nil {
			return errors.Wrap(nil, err)
		}

		pk, _ := ioutil.ReadAll((privKey))

		cert, err = tls.X509KeyPair([]byte(cc), []byte(pk))
		if err != nil {
			return errors.Wrap(nil, err)
		}

		cfg.MQTT.Cert = cert
		cfg.MQTT.CA = caByte

	}
	return nil
}

func startHTTPService(svc export.Service, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Export service started, exposed port %s", p))
	errs <- http.ListenAndServe(p, api.MakeHandler(svc))
}
