# export
Mainflux Export service that sends messages from one Mainflux cloud to another via MQTT

## Install
Get the code:

```
go get github.com/mainflux/export
cd $GOPATH/github.com/mainflux/export
```

Make:
```
make
```

## Usage
### Config
Export configuration is kept in `../configs/config.toml`.


```
channels = "../docker/channels.toml"

[exp]
  log_level = "debug"
  nats = "nats://127.0.0.1:4222"
  port = "8170"

[mqtt]
  channel = "channel/4c66a785-1900-4844-8caa-56fb8cfd61eb/messages"
  username = "4a437f46-da7b-4469-96b7-be754b5e8d36"
  password = "ac6b57e0-9b70-45d6-94c8-e67ac9086165"
  ca = "ca.crt"
  cert = "thing.crt"
  mtls = "false"
  priv_key = "thing.key"
  retain = "false"
  skip_tls_ver = "false"
  url = "tcp://mainflux.com:1883"
```
to run it edit configs/config.toml and change `channel`, `username`,`password` and `url`
 * `username` is actually thing id in mainflux cloud instance
 * `password` is thing key
 * `channel` is mqtt topic where to publish mqtt data ( `channel/<channel_id>/messages` is format of mainflux mqtt topic)

in order for export service to listen on mainflux nats deployed on the same machine NATS port must be exposed
edit docker-compose.yml of mainflux (github.com/mainflux/mainflux/docker/docker-compose.yml )
nats section must look like below
```
  nats:
    image: nats:1.3.0
    container_name: mainflux-nats
    restart: on-failure
    networks:
      - mainflux-base-net
    ports:
      - 4222:4222
```
  
## Environmet variables

Service will look for config.toml first if not found it will  be configured 
with env variables and new config.toml will be saved with values populated from env vars.
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
| MF_EXPORT_MQTT_RETAIN         | MQTT retain                                                   | false                 |
| MF_EXPORT_CONF_PATH           | Configuration file                                            | config.toml           |
| MF_EXPORT_CHANNELS_CONFIG     | Channels to which will export service listen to               | channels.toml         |

to change values be sure that there is no config.toml as this is default


