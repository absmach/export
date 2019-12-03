// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt_test

import (
	"fmt"
	"os"
	"testing"

	dockertest "gopkg.in/ory-am/dockertest.v3"
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	cfg := []string{
		"INFLUXDB_USER=test",
		"INFLUXDB_USER_PASSWORD=test",
		"INFLUXDB_DB=test",
	}
	container, err := pool.Run("influxdb", "1.6.4-alpine", cfg)
	if err != nil {
		testLog.Error(fmt.Sprintf("Could not start container: %s", err))
	}

	port = container.GetPort("8086/tcp")
	clientCfg.Addr = fmt.Sprintf("http://localhost:%s", port)

	// if err := pool.Retry(func() error {
	// 	client, err = influxdb.NewHTTPClient(clientCfg)
	// 	_, _, err = client.Ping(5 * time.Millisecond)
	// 	return err
	// }); err != nil {
	// 	testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	// }

	code := m.Run()

	if err := pool.Purge(container); err != nil {
		testLog.Error(fmt.Sprintf("Could not purge container: %s", err))
	}

	os.Exit(code)
}
