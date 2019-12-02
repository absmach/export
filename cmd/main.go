// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/export/internal/app/export/api"
	"github.com/mainflux/export/internal/pkg/config"
	exmqtt "github.com/mainflux/export/internal/pkg/mqtt"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
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
	defConfigFile     = "config.toml"

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
	envMqttRetain     = "MF_EXPORT_RETAINS"
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

	client, err := mqttConnect(svcName, cfg, logger)
	if err != nil {
		log.Fatalf(err.Error())
	}

	repo := exmqtt.New(client, cfg, logger)

	counter, latency := makeMetrics()
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(repo, counter, latency)
	svc := export.New(nc, repo, nil, nil, nil, logger)
	//svc := export.New(nc, repo, nil, cfg.Channels, nil, logger)
	if err := svc.Start(svcName); err != nil {
		logger.Error(fmt.Sprintf("Failed to start exporte service: %s", err))
		os.Exit(1)
	}

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

func viperSave(configFile string, cfg map[string]string) error {

	for key, val := range cfg {
		viper.Set(key, val)
	}

	viper.SetConfigFile(configFile)
	viper.WriteConfig()

	return nil
}

func viperRead(configFile string) (export.Config, error) {
	viper.SetConfigFile(configFile)
	cfg := export.Config{}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err.Error())
		return cfg, fmt.Errorf("Configuration file error: %s", err)
	}
	viper.Unmarshal(&cfg)
	err := loadCertificate(&cfg)

	if err != nil {
		return cfg, err
	}
	return cfg, nil

}

func loadConfigs() (export.Config, error) {
	configFile := mainflux.Env(envConfigFile, defConfigFile)
	cfg := config.New(nil, nil, nil, configFile)
	err := cfg.Read()
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

		sc : = config.Server {
			NatsURL : mainflux.Env(envNatsURL, defNatsURL),
			LogLevel : mainflux.Env(envLogLevel, defLogLevel),
			Port : mainflux.Env(envPort, defPort),
		}

		mc := config.MQTT {
			Host  : mainflux.Env(envMqttHost, defMqttHost),
			Password : mainflux.Env(envMqttPassword, defMqttPassword),
			Username : mainflux.Env(envMqttUsername, defMqttUsername),
			
			Retain : mqttRetain,
			QoS : QoS,
			MTLS : mqttMTLS,
			SkipTLSVer : mqttSkipTLSVer,

			CAPath : mainflux.Env(envMqttCA, defMqttCA),
			ClientCertPath : mainflux.Env(envMqttCert, defMqttCert),
			PrivKeyPath : mainflux.Env(envMqttPrivKey, defMqttPrivKey),
		}
		routes := []config.Route {
			MqttTopic : mainflux.Env(envMqttChannel, defMqttChannel),
			NatsTopic : "*"
		}
		rc := config.Routes{routes}
		cfg := config.New(sc, rc, mc, configFile )
		err = loadCertificate(&cfg)
		if err != nil {
			return cfg, err
		}

		log.Println(fmt.Sprintf("Configuration loaded from environment, initial %s saved", configFile))
		return cfg, nil
	}
	err = loadCertificate(&cfg)
	if err != nil {
		return cfg, err
	}
	log.Println(fmt.Sprintf("Configuration loaded from file %s", configFile))
	return cfg, nil
}

func loadCertificate(cfg *export.Config) error {

	caByte := []byte{}
	cert := tls.Certificate{}
	if cfg.Mqtt.MTLS {
		caFile, err := os.Open(cfg.Mqtt.CAPath)
		defer caFile.Close()
		if err != nil {
			return err
		}
		caByte, _ = ioutil.ReadAll(caFile)

		clientCert, err := os.Open(cfg.Mqtt.CertPath)
		defer clientCert.Close()
		if err != nil {
			return err
		}
		cc, _ := ioutil.ReadAll(clientCert)

		privKey, err := os.Open(cfg.Mqtt.PrivKeyPath)
		defer clientCert.Close()
		if err != nil {
			return err
		}

		pk, _ := ioutil.ReadAll((privKey))

		cert, err = tls.X509KeyPair([]byte(cc), []byte(pk))
		if err != nil {
			return err
		}

		cfg.Mqtt.Cert = cert
		cfg.Mqtt.CA = caByte

	}
	return nil
}

// type channels struct {
// 	List []string `toml:"filter"`
// }

// type chanConfig struct {
// 	Channels channels `toml:"channels"`
// }

// func loadChansConfig(chanConfigPath string) map[string]bool {
// 	data, err := ioutil.ReadFile(chanConfigPath)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	var chanCfg chanConfig
// 	if err := toml.Unmarshal(data, &chanCfg); err != nil {
// 		log.Fatal(err)
// 	}

// 	chans := map[string]bool{}
// 	for _, ch := range chanCfg.Channels.List {
// 		chans[ch] = true
// 	}

// 	return chans
// }

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

func mqttConnect(name string, conf export.Config, logger logger.Logger) (mqtt.Client, error) {
	conn := func(client mqtt.Client) {
		logger.Info(fmt.Sprintf("Client %s connected", name))
	}

	lost := func(client mqtt.Client, err error) {
		logger.Info(fmt.Sprintf("Client %s disconnected", name))
	}

	opts := mqtt.NewClientOptions().
		AddBroker(conf.Mqtt.MqttHost).
		SetClientID(name).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(conn).
		SetConnectionLostHandler(lost)

	if conf.Mqtt.MqttUsername != "" && conf.Mqtt.MqttPassword != "" {
		opts.SetUsername(conf.Mqtt.MqttUsername)
		opts.SetPassword(conf.Mqtt.MqttPassword)
	}

	if conf.Mqtt.MqttMTLS {
		cfg := &tls.Config{
			InsecureSkipVerify: conf.Mqtt.MqttSkipTLSVer,
		}

		if conf.Mqtt.MqttCA != nil {
			cfg.RootCAs = x509.NewCertPool()
			cfg.RootCAs.AppendCertsFromPEM(conf.Mqtt.MqttCA)
		}
		if conf.Mqtt.MqttCert.Certificate != nil {
			cfg.Certificates = []tls.Certificate{conf.Mqtt.MqttCert}
		}

		cfg.BuildNameToCertificate()
		opts.SetTLSConfig(cfg)
		opts.SetProtocolVersion(4)
	}
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		logger.Error(fmt.Sprintf("Client %s had error connecting to the broker: %s\n", name, token.Error().Error()))
		return nil, token.Error()
	}
	return client, nil
}
