# export
Mainflux Export service that sends messages from one Mainflux cloud to another via MQTT


## Configuration

Service will look for config.toml first if not found it will  be configured 
with env variables and new config.toml will be saved.
The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                      | Description                                                   | Default               |
|-------------------------------|---------------------------------------------------------------|-----------------------|
| MF_NATS_URL                   | Nats url                                                      | localhost:4222        |
| MF_EXPORT_MQTT_HOST           | Mqtt url where to export                                      | tcp://localhost:1883  |
| MF_EXPORT_MQTT_USERNAME       | MQTT username, thing id in case of mainflux                   |                       | 
| MF_EXPORT_MQTT_PASSWORD       | MQTT password, thing key in case of mainflux                  |                       |
| MF_EXPORT_MQTT_CHANNEL        | MQTT channel where to publish                                 |                       |
| MF_EXPORT_MQTT_SKIP_TLS       | Skip tls verification                                         | true                  |
| MF_EXPORT_MQTT_MTLS           | Use MTLS for authentication                                   | false                 |
| MF_EXPORT_MQTT_CA             | CA for tls                                                    | ca.crt                |
| MF_EXPORT_MQTT_CLIENT_CERT    | Client cert for authentication in case when MTLS = true       | thing.crt             |
| MF_EXPORT_MQTT_CLIENT_PK      | Client key for authentication in case when MTLS = true        | thing.key             |
| MF_EXPORT_MQTT_QOS            | MQTT QOS                                                      | 0                     |
| MF_EXPORT_RETAINS             | MQTT retain                                                   | false                 |
| MF_EXPORT_CONF_PATH           | Configuration file                                            | config.toml           |
| MF_EXPORT_CHANNELS_CONFIG     | Channels to which will export service listen to               | channels.toml         |

to change values be sure that there is no config.toml as this is default

example of config file

```
channels = "../docker/channels.toml"

[exp]
  log_level = "debug"
  nats = "nats://127.0.0.1:4222"
  port = "8170"

[mqtt]
  ca = "ca.crt"
  cert = "thing.crt"
  channel = "4c66a785-1900-4844-8caa-56fb8cfd61eb"
  mtls = "false"
  password = "ac6b57e0-9b70-45d6-94c8-e67ac9086165"
  priv_key = "thing.key"
  retain = "false"
  skip_tls_ver = "false"
  url = "tcp://localhost:1883"
  username = "4a437f46-da7b-4469-96b7-be754b5e8d36"
```