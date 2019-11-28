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

	"github.com/BurntSushi/toml"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/export/writers/api"
	"github.com/mainflux/export/writers/export"
	writer "github.com/mainflux/export/writers/export/writer"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

const (
	svcName           = "exporter"
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
	defMqttQoS        = 0
	defMqttRetain     = false
	defMqttCert       = "thing.cert"
	defMqttPrivKey    = "thing.key"
	defConfPath       = "config.toml"
	defChanCfgPath    = "/config/channels.toml"

	envNatsURL  = "MF_NATS_URL"
	envLogLevel = "MF_EXPORTER_LOG_LEVEL"
	envPort     = "MF_EXPORTER_PORT"

	envMqttHost       = "MF_EXPORTER_MQTT_HOST"
	envMqttUsername   = "MF_EXPORTER_MQTT_USERNAME"
	envMqttPassword   = "MF_EXPORTER_MQTT_PASSWORD"
	envMqttChannel    = "MF_EXPORTER_MQTT_CHANNEL"
	envMqttSkipTLSVer = "MF_EXPORTER_MQTT_SKIP_TLS"
	envMqttMTLS       = "MF_EXPORTER_MQTT_MTLS"
	envMqttCA         = "MF_EXPORTER_MQTT_CA"
	envMqttQoS        = "MF_EXPORTER_MQTT_QOS"
	envRetain         = "MF_EXPORTER_RETAINS"
	envMqttCert       = "MF_EXPORTER_MQTT_CLIENT_CERT"
	envMqttPrivKey    = "MF_EXPORTER_MQTT_CLIENT_PK"
	envConfPath       = "MF_EXPORTER_CONF_PATH"
	envChanCfgPath    = "MF_EXPORTER_CHANNELS_CONFIG"

	keyMqttMTls       = "mqtt.mtls"
	keyMqttSkipTLS    = "mqtt.skip_tls_ver"
	keyMqttUrl        = "mqtt.url"
	keyMqttClientCert = "mqtt.cert"
	keyMqttPrivKey    = "mqtt.priv_key"
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

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

	client, err := mqttConnect(svcName, cfg, logger)
	if err != nil {
		log.Fatalf(err.Error())
	}

	repo := export.New(client, cfg, logger)

	counter, latency := makeMetrics()
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(repo, counter, latency)
	w := writer.New(nc, repo, nil, cfg.Channels, nil, logger)
	if err := w.Start(svcName); err != nil {
		logger.Error(fmt.Sprintf("Failed to start exporte service: %s", err))
		os.Exit(1)
	}

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go startHTTPService(cfg.Port, logger, errs)

	err = <-errs
	logger.Error(fmt.Sprintf("export writer service terminated: %s", err))
}

func viperRead(configFile string) (export.Config, error) {
	viper.SetConfigFile(configFile)
	cfg := export.Config{}

	if err := viper.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("Configuration file error: %s", err)
	}

	viperCfg := map[string]string{
		keyMqttMTls:       "mqtt.mtls",
		keyMqttSkipTLS:    "mqtt.skip_tls_ver",
		keyMqttUrl:        "mqtt.url",
		keyMqttClientCert: "mqtt.cert",
		keyMqttPrivKey:    "mqtt.priv_key",
		keyMqttCA:         "mqtt.ca",
		keyMqttPassword:   "mqtt.password",
		keyMqttUsername:   "mqtt.username",
		keyMqttChannel:    "mqtt.channel",
		keyChanCfg:        "channels",
	}

	for key := range viperCfg {
		val := viper.GetString(key)
		viperCfg[key] = val
	}
	CA := viperCfg[keyMqttCA]
	ClientCert := viperCfg[keyMqttClientCert]
	PrivKey := viperCfg[keyMqttPrivKey]

	err := loadCertificate(&cfg, CA, ClientCert, PrivKey)
	if err != nil {
		return cfg, err
	}
	return cfg, nil

}

func loadConfigs() (export.Config, error) {
	chanCfgPath := mainflux.Env(envChanCfgPath, defChanCfgPath)
	confPath := mainflux.Env(envConfPath, defConfPath)
	cfg, err := viperRead(confPath)
	if err != nil {
		mqttSkipTLSVer, err := strconv.ParseBool(mainflux.Env(envMqttSkipTLSVer, defMqttSkipTLSVer))
		if err != nil {
			return export.Config{}, err
		}
		mqttMTLS, err := strconv.ParseBool(mainflux.Env(envMqttMTLS, defMqttMTLS))
		if err != nil {
			return export.Config{}, err
		}

		cfg = export.Config{
			NatsURL:        mainflux.Env(envNatsURL, defNatsURL),
			LogLevel:       mainflux.Env(envLogLevel, defLogLevel),
			Port:           mainflux.Env(envPort, defPort),
			MqttHost:       mainflux.Env(envMqttHost, defMqttHost),
			MqttChannel:    mainflux.Env(envMqttChannel, defMqttChannel),
			MqttPassword:   mainflux.Env(envMqttPassword, defMqttPassword),
			MqttUsername:   mainflux.Env(envMqttUsername, defMqttUsername),
			Channels:       loadChansConfig(chanCfgPath),
			MqttMTLS:       mqttMTLS,
			MqttSkipTLSVer: mqttSkipTLSVer,
		}
		CA := mainflux.Env(envMqttCA, defMqttCA)
		ClientCert := mainflux.Env(envMqttCert, defMqttCert)
		PrivKey := mainflux.Env(envMqttPrivKey, defMqttPrivKey)

		err = loadCertificate(&cfg, CA, ClientCert, PrivKey)
		if err != nil {
			return cfg, err
		}

		return cfg, nil
	}

	return cfg, nil
}

func loadCertificate(cfg *export.Config, CA, ClientCert, PrivKey string) error {

	caByte := []byte{}
	cert := tls.Certificate{}
	if cfg.MqttMTLS {
		caFile, err := os.Open(CA)
		defer caFile.Close()
		if err != nil {
			return err
		}
		caByte, _ = ioutil.ReadAll(caFile)

		clientCert, err := os.Open(ClientCert)
		defer clientCert.Close()
		if err != nil {
			return err
		}
		cc, _ := ioutil.ReadAll(clientCert)

		privKey, err := os.Open(PrivKey)
		defer clientCert.Close()
		if err != nil {
			return err
		}

		pk, _ := ioutil.ReadAll((privKey))

		cert, err = tls.X509KeyPair([]byte(cc), []byte(pk))
		if err != nil {
			return err
		}

		cfg.MqttCert = cert
		cfg.MqttCA = caByte

	}
	return nil
}

type channels struct {
	List []string `toml:"filter"`
}

type chanConfig struct {
	Channels channels `toml:"channels"`
}

func loadChansConfig(chanConfigPath string) map[string]bool {
	data, err := ioutil.ReadFile(chanConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	var chanCfg chanConfig
	if err := toml.Unmarshal(data, &chanCfg); err != nil {
		log.Fatal(err)
	}

	chans := map[string]bool{}
	for _, ch := range chanCfg.Channels.List {
		chans[ch] = true
	}

	return chans
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

func startHTTPService(port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Exporter service started, exposed port %s", p))
	errs <- http.ListenAndServe(p, api.MakeHandler(svcName))
}

func mqttConnect(name string, conf export.Config, logger logger.Logger) (mqtt.Client, error) {
	conn := func(client mqtt.Client) {
		logger.Info(fmt.Sprintf("Client %s connected", name))
	}

	lost := func(client mqtt.Client, err error) {
		logger.Info(fmt.Sprintf("Client %s disconnected", name))
	}

	opts := mqtt.NewClientOptions().
		AddBroker(conf.MqttHost).
		SetClientID(name).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(conn).
		SetConnectionLostHandler(lost)

	if conf.MqttUsername != "" && conf.MqttPassword != "" {
		opts.SetUsername(conf.MqttUsername)
		opts.SetPassword(conf.MqttPassword)
	}

	if conf.MqttMTLS {
		cfg := &tls.Config{
			InsecureSkipVerify: conf.MqttSkipTLSVer,
		}

		if conf.MqttCA != nil {
			cfg.RootCAs = x509.NewCertPool()
			cfg.RootCAs.AppendCertsFromPEM(conf.MqttCA)
		}
		if conf.MqttCert.Certificate != nil {
			cfg.Certificates = []tls.Certificate{conf.MqttCert}
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
