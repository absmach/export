// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mainflux/export/pkg/config"
	exp "github.com/mainflux/export/pkg/config"
	"github.com/mainflux/export/pkg/export"
	"github.com/mainflux/export/pkg/export/api"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	nats "github.com/nats-io/nats.go"
)

const (
	svcName           = "export"
	defBrokerURL      = nats.DefaultURL
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
	defMqttRetain     = "false"
	defMqttCert       = "thing.cert"
	defMqttPrivKey    = "thing.key"
	defConfigFile     = "../configs/config.toml"

	defCacheURL  = "localhost:6379"
	defCachePass = ""
	defCacheDB   = "0"

	envBrokerURL = "MF_BROKER_URL"
	envLogLevel  = "MF_EXPORT_LOG_LEVEL"
	envPort      = "MF_EXPORT_PORT"

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
	envConfigFile     = "MF_EXPORT_CONFIG_FILE"

	envCacheURL  = "MF_EXPORT_CACHE_URL"
	envCachePass = "MF_EXPORT_CACHE_PASS"
	envCacheDB   = "MF_EXPORT_CACHE_DB"

	heartbeatSubject = "heartbeat"
	service          = "service"
)

func main() {
	ctx := context.Background()
	cfg, err := loadConfigs()
	if err != nil {
		log.Fatalf(err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.Server.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	pubsub, err := brokers.NewPubSub(cfg.Server.BrokerURL, "", logger)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to connect to Broker: %s %s", err, cfg.Server.BrokerURL))
	}
	defer pubsub.Close()

	svc, err := export.New(cfg, logger, pubsub)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create service :%s", err))
		return
	}
	if err := svc.Start(svcName); err != nil {
		logger.Error(fmt.Sprintf("Failed to start service %s", err))
		return
	}
	svc.Subscribe(ctx)

	// Publish heartbeat
	ticker := time.NewTicker(10000 * time.Millisecond)
	go func() {
		subject := fmt.Sprintf("%s.%s.%s", heartbeatSubject, svcName, service)
		for range ticker.C {
			if err := pubsub.Publish(ctx, subject, &messaging.Message{Channel: subject}); err != nil {
				logger.Error(fmt.Sprintf("Failed to publish heartbeat, %s", err))
			}
		}
	}()

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go startHTTPService(svc, cfg.Server.Port, logger, errs)

	err = <-errs
	logger.Error(fmt.Sprintf("export writer service terminated: %s", err))
}

func loadConfigs() (exp.Config, error) {
	configFile := mainflux.Env(envConfigFile, defConfigFile)
	cfg, err := config.ReadFile(configFile)
	if err != nil {
		mqttSkipTLSVer, err := strconv.ParseBool(mainflux.Env(envMqttSkipTLSVer, defMqttSkipTLSVer))
		if err != nil {
			mqttSkipTLSVer = false
		}
		mqttMTLS, err := strconv.ParseBool(mainflux.Env(envMqttMTLS, defMqttMTLS))
		if err != nil {
			mqttMTLS = false
		}
		mqttRetain, err := strconv.ParseBool(mainflux.Env(envMqttRetain, defMqttRetain))
		if err != nil {
			mqttRetain = false
		}

		q, err := strconv.ParseInt(mainflux.Env(envMqttQoS, defMqttQoS), 10, 64)
		if err != nil {
			q = 0
		}
		QoS := int(q)

		sc := exp.Server{
			BrokerURL: mainflux.Env(envBrokerURL, defBrokerURL),
			LogLevel:  mainflux.Env(envLogLevel, defLogLevel),
			Port:      mainflux.Env(envPort, defPort),
			CachePass: mainflux.Env(envCachePass, defCachePass),
			CacheURL:  mainflux.Env(envCacheURL, defCacheURL),
			CacheDB:   mainflux.Env(envCacheDB, defCacheDB),
		}

		mc := exp.MQTT{
			Host:     mainflux.Env(envMqttHost, defMqttHost),
			Password: mainflux.Env(envMqttPassword, defMqttPassword),
			Username: mainflux.Env(envMqttUsername, defMqttUsername),

			Retain:     mqttRetain,
			QoS:        QoS,
			MTLS:       mqttMTLS,
			SkipTLSVer: mqttSkipTLSVer,

			CAPath:            mainflux.Env(envMqttCA, defMqttCA),
			ClientCertPath:    mainflux.Env(envMqttCert, defMqttCert),
			ClientPrivKeyPath: mainflux.Env(envMqttPrivKey, defMqttPrivKey),
		}
		mqttChannel := mainflux.Env(envMqttChannel, defMqttChannel)
		mqttTopic := export.Channels + "/" + mqttChannel + "/" + export.Messages
		natsTopic := export.NatsSub
		rc := []exp.Route{{
			MqttTopic: mqttTopic,
			NatsTopic: natsTopic,
		}}

		cfg := exp.Config{
			Server: sc,
			Routes: rc,
			MQTT:   mc,
			File:   configFile,
		}
		mqtt, err := loadCertificate(cfg.MQTT)
		cfg.MQTT = mqtt
		if err != nil {
			return cfg, err
		}

		if err := exp.Save(cfg); err != nil {
			log.Printf("Failed to save %s\n", err)
		}
		log.Printf("Configuration loaded from environment, initial %s saved\n", configFile)
		return cfg, nil
	}
	mqtt, err := loadCertificate(cfg.MQTT)
	if err != nil {
		return cfg, err
	}
	cfg.MQTT = mqtt
	log.Printf("Configuration loaded from file %s\n", configFile)
	return cfg, nil
}

func loadCertificate(cfg exp.MQTT) (exp.MQTT, error) {
	var caByte []byte
	var cc []byte
	var pk []byte
	if !cfg.MTLS {
		return cfg, nil
	}

	caFile, err := os.Open(cfg.CAPath)
	if err != nil {
		return cfg, errors.New(err.Error())
	}
	defer caFile.Close()
	caByte, _ = io.ReadAll(caFile)

	if cfg.ClientCertPath != "" {
		clientCert, err := os.Open(cfg.ClientCertPath)
		if err != nil {
			return cfg, errors.New(err.Error())
		}
		defer clientCert.Close()
		cc, err = io.ReadAll(clientCert)
		if err != nil {
			return cfg, err
		}
	}

	if len(cc) == 0 && cfg.ClientCert != "" {
		cc = []byte(cfg.ClientCert)
	}

	if cfg.ClientPrivKeyPath != "" {
		privKey, err := os.Open(cfg.ClientPrivKeyPath)
		if err != nil {
			return cfg, errors.New(err.Error())
		}
		defer privKey.Close()
		pk, err = io.ReadAll((privKey))
		if err != nil {
			return cfg, err
		}
	}

	if len(pk) == 0 && cfg.ClientCertKey != "" {
		pk = []byte(cfg.ClientCertKey)
	}

	if len(pk) == 0 || len(cc) == 0 {
		return cfg, errors.New("failed loading client certificate")
	}

	cert, err := tls.X509KeyPair([]byte(cc), []byte(pk))
	if err != nil {
		return cfg, errors.New(err.Error())
	}

	cfg.TLSCert = cert
	cfg.CA = caByte

	return cfg, nil
}

func startHTTPService(svc export.Service, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Export service started, exposed port %s", p))
	errs <- http.ListenAndServe(p, api.MakeHandler(svc))
}
