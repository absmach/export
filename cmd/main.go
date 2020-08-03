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
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis"
	"github.com/mainflux/export/pkg/config"
	exp "github.com/mainflux/export/pkg/config"
	"github.com/mainflux/export/pkg/export"
	"github.com/mainflux/export/pkg/export/api"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
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

	defCacheURL  = "localhost:6379"
	defCachePass = ""
	defCacheDB   = "0"

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
	envConfigFile     = "MF_EXPORT_CONFIG_FILE"

	envCacheURL  = "MF_EXPORT_CACHE_URL"
	envCachePass = "MF_EXPORT_CACHE_PASS"
	envCacheDB   = "MF_EXPORT_CACHE_DB"

	heartbeatSubject = "heartbeat"
	service          = "service"
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

	svc, err := export.New(cfg, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create service :%s", err))
		os.Exit(1)
	}
	if err := svc.Start(svcName); err != nil {
		logger.Error(fmt.Sprintf("Failed to start service %s", err))
		os.Exit(1)
	}
	svc.Subscribe(nc)

	// Publish heartbeat
	ticker := time.NewTicker(10000 * time.Millisecond)
	go func() {
		subject := fmt.Sprintf("%s.%s.%s", heartbeatSubject, svcName, service)
		for range ticker.C {
			if err := nc.Publish(subject, []byte{}); err != nil {
				logger.Error(fmt.Sprintf("Failed to publish heartbeat, %s", err))
			}
		}
	}()

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
		mqttRetain, err := strconv.ParseBool(mainflux.Env(envMqttMTLS, defMqttMTLS))
		if err != nil {
			mqttRetain = false
		}

		q, err := strconv.ParseInt(mainflux.Env(envMqttQoS, defMqttQoS), 10, 64)
		if err != nil {
			q = 0
		}
		QoS := int(q)

		sc := exp.Server{
			NatsURL:   mainflux.Env(envNatsURL, defNatsURL),
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
			log.Println(fmt.Sprintf("Failed to save %s", err))
		}
		log.Println(fmt.Sprintf("Configuration loaded from environment, initial %s saved", configFile))
		return cfg, nil
	}
	mqtt, err := loadCertificate(cfg.MQTT)
	if err != nil {
		return cfg, err
	}
	cfg.MQTT = mqtt
	log.Println(fmt.Sprintf("Configuration loaded from file %s", configFile))
	return cfg, nil
}

func loadCertificate(cfg exp.MQTT) (exp.MQTT, error) {
	var caByte []byte
	var cc []byte
	var pk []byte
	cert := tls.Certificate{}
	if cfg.MTLS == false {
		return cfg, nil
	}

	caFile, err := os.Open(cfg.CAPath)
	if err != nil {
		return cfg, errors.New(err.Error())
	}
	defer caFile.Close()
	caByte, _ = ioutil.ReadAll(caFile)

	if cfg.ClientCertPath != "" {
		clientCert, err := os.Open(cfg.ClientCertPath)
		if err != nil {
			return cfg, errors.New(err.Error())
		}
		defer clientCert.Close()
		cc, err = ioutil.ReadAll(clientCert)
		if err != nil {
			return cfg, err
		}
	}

	if len(cc) == 0 && cfg.ClientCert != "" {
		cc = []byte(cfg.ClientCert)
	}

	if cfg.ClientPrivKeyPath != "" {
		privKey, err := os.Open(cfg.ClientPrivKeyPath)
		defer privKey.Close()
		if err != nil {
			return cfg, errors.New(err.Error())
		}
		pk, err = ioutil.ReadAll((privKey))
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

	cert, err = tls.X509KeyPair([]byte(cc), []byte(pk))
	if err != nil {
		return cfg, errors.New(err.Error())
	}

	cfg.TLSCert = cert
	cfg.CA = caByte

	return cfg, nil
}

func makeMetrics() (*kitprometheus.Counter, *kitprometheus.Summary) {
	counter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "export",
		Subsystem: "message_writer",
		Name:      "request_count",
		Help:      "Number of database inserts.",
	}, []string{"method"})

	latency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "export",
		Subsystem: "message_writer",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of inserts in microseconds.",
	}, []string{"method"})

	return counter, latency
}

func startHTTPService(svc export.Service, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Export service started, exposed port %s", p))
	errs <- http.ListenAndServe(p, api.MakeHandler(svc))
}

func connectToRedis(cacheURL, cachePass string, cacheDB string, logger logger.Logger) *redis.Client {
	db, err := strconv.Atoi(cacheDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to cache: %s", err))
		return nil
	}

	return redis.NewClient(&redis.Options{
		Addr:     cacheURL,
		Password: cachePass,
		DB:       db,
	})
}
