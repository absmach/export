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
	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/export/internal/app/export/api"
	"github.com/mainflux/export/internal/pkg/messages"
	"github.com/mainflux/export/pkg/config"
	exp "github.com/mainflux/export/pkg/config"
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
	envConfigFile     = "MF_EXPORT_CONF_PATH"

	envCacheURL  = "MF_EXPORT_CACHE_URL"
	envCachePass = "MF_EXPORT_CACHE_PASS"
	envCacheDB   = "MF_EXPORT_CACHE_DB"

	heartbeatSubject = "heartbeat"
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

	redisClient := connectToRedis(cfg.Server.CacheURL, cfg.Server.CachePass, cfg.Server.CacheDB, logger)
	msgCache := messages.NewRedisCache(redisClient)

	svc := export.New(cfg, msgCache, logger)
	if err := svc.Start(svcName); err != nil {
		logger.Error(fmt.Sprintf("Failed to start service %s", err))
		os.Exit(1)
	}
	svc.Subscribe(nc)

	// Publish heartbeat
	ticker := time.NewTicker(10000 * time.Millisecond)
	go func() {
		subject := fmt.Sprintf("%s.%s", heartbeatSubject, "export")
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

			CAPath:      mainflux.Env(envMqttCA, defMqttCA),
			CertPath:    mainflux.Env(envMqttCert, defMqttCert),
			PrivKeyPath: mainflux.Env(envMqttPrivKey, defMqttPrivKey),
		}
		mqttTopic := mainflux.Env(envMqttChannel, defMqttChannel)
		natsTopic := "*"
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

func loadCertificate(cfg exp.MQTT) (exp.MQTT, errors.Error) {

	caByte := []byte{}
	cert := tls.Certificate{}
	if cfg.MTLS {
		caFile, err := os.Open(cfg.CAPath)
		if err != nil {
			return cfg, errors.New(err.Error())
		}
		defer caFile.Close()
		caByte, _ = ioutil.ReadAll(caFile)

		clientCert, err := os.Open(cfg.CertPath)
		if err != nil {
			return cfg, errors.New(err.Error())
		}
		defer clientCert.Close()
		cc, _ := ioutil.ReadAll(clientCert)

		privKey, err := os.Open(cfg.PrivKeyPath)
		defer clientCert.Close()
		if err != nil {
			return cfg, errors.New(err.Error())
		}

		pk, _ := ioutil.ReadAll((privKey))

		cert, err = tls.X509KeyPair([]byte(cc), []byte(pk))
		if err != nil {
			return cfg, errors.New(err.Error())
		}

		cfg.Cert = cert
		cfg.CA = caByte

	}
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
