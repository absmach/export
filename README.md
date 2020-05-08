# Export
Mainflux Export service can sends message from one Mainflux cloud to another via MQTT, or it can send messages from edge gateway.
Export service is subscribed to local message bus and connected to mqtt broker.  
Messages collected on local message bus are redirected to cloud.
Export service can store messages into `Redis` streams when connection is lost and upon connection reestablishment it consumes messages from stream and send it to cloud.


## Install
Get the code:

```bash
go get github.com/mainflux/export
cd $GOPATH/github.com/mainflux/export
```

Make:
```bash
make
```

## Usage

```bash
cd build
./mainflux-export
```

## Configuration
By default it will look for config file at [`../configs/config.toml`][conftoml] if no env vars are specified.  

```toml
[exp]
  cache_pass = ""
  cache_url = "localhost:6379"
  cache_db = "0"
  log_level = "debug"
  nats = "localhost:4222"
  port = "8170"

[mqtt]
  username = "<thing_id>"
  password = "<thing_password>"
  ca = "ca.crt"
  cert = "thing.crt"
  mtls = "false"
  priv_key = "thing.key"
  retain = "false"
  skip_tls_ver = "false"
  url = "tcp://mainflux.com:1883"

[[routes]]
  mqtt_topic = "channel/<channel_id>/messages"
  subtopic = "subtopic"
  nats_topic = ".>"
  type = "plain"
  workers = 10
```
### Http port
- `port` - HTTP port where status of service can be obtained
```bash
curl -X GET http://localhost:8170/version
{"service":"export","version":"0.0.1"}%
``` 
### Redis connection
To configure `Redis` connection settings `cache_url`, `cache_pass`, `cache_db` in `config.toml` are used.

### MQTT connection

To establish connection to MQTT broker following settings are needed:
- `username` - Mainflux <thing_id>
- `password` - Mainflux <thing_key>
- `url` - url of MQTT broker

Additionally, you will need MQTT client certificates if you enable mTLS. To obtain certificates `ca.crt`, `thing.crt` and key `thing.key` follow instructions [here](https://mainflux.readthedocs.io/en/latest/authentication/#mutual-tls-authentication-with-x509-certificates).

  
### Routes 
Routes are being used for specifying which subscriber topic(subject) goes to which publishing topic.
Currently only mqtt is supported for publishing. To match Mainflux requirements `mqtt_topic` must contain `channel/<channel_id>/messages`, additional subtopics can be appended.
- `export` service will be subscribed to NATS subject `export.<nats_topic>`
- messages will be published to MQTT topic `<mqtt_topic>/<subtopic>/<nats_subject>, where dots in nats_subject are replaced with '/'
- workers control number of workers that will be used for message forwarding.

to run it edit configs/config.toml and change `channel`, `username`, `password` and `url`
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
  
## Environment variables

Service will look for `config.toml` first and if not found it will be configured with env variables and new `config.toml` will be saved with values populated from env vars.  
The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

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

for values to take effect make sure that there is no `MF_EXPORT_CONF` file.

If you run with environment variables you can create config file:
```bash
MF_EXPORT_PORT=8178 \
MF_EXPORT_LOG_LEVEL=debug \
MF_EXPORT_MQTT_HOST=tcp://localhost:1883 \
MF_EXPORT_MQTT_USERNAME=88529fb2-6c1e-4b60-b9ab-73b5d89f7404 \
MF_EXPORT_MQTT_PASSWORD=ac6b57e0-9b70-45d6-94c8-e67ac9086165 \
MF_EXPORT_MQTT_CHANNEL=4c66a785-1900-4844-8caa-56fb8cfd61eb \
MF_EXPORT_MQTT_SKIP_TLS=true \
MF_EXPORT_MQTT_MTLS=false \
MF_EXPORT_MQTT_CA=ca.crt \
MF_EXPORT_MQTT_CLIENT_CERT=thing.crt \
MF_EXPORT_MQTT_CLIENT_PK=thing.key \
MF_EXPORT_CONF_PATH=export.toml \
../build/mainflux-export&
```
## How to save config via agent

```
mosquitto_pub -u <thing_id> -P <thing_key> -t channels/<control_ch_id>/messages/req -h localhost -p 18831  -m  "[{\"bn\":\"1:\", \"n\":\"config\", \"vs\":\"config.toml, RmlsZSA9ICIuLi9jb25maWdzL2NvbmZpZy50b21sIgoKW2V4cF0KICBsb2dfbGV2ZWwgPSAiZGVidWciCiAgbmF0cyA9ICJuYXRzOi8vMTI3LjAuMC4xOjQyMjIiCiAgcG9ydCA9ICI4MTcwIgoKW21xdHRdCiAgY2FfcGF0aCA9ICJjYS5jcnQiCiAgY2VydF9wYXRoID0gInRoaW5nLmNydCIKICBjaGFubmVsID0gIiIKICBob3N0ID0gInRjcDovL2xvY2FsaG9zdDoxODgzIgogIG10bHMgPSBmYWxzZQogIHBhc3N3b3JkID0gImFjNmI1N2UwLTliNzAtNDVkNi05NGM4LWU2N2FjOTA4NjE2NSIKICBwcml2X2tleV9wYXRoID0gInRoaW5nLmtleSIKICBxb3MgPSAwCiAgcmV0YWluID0gZmFsc2UKICBza2lwX3Rsc192ZXIgPSBmYWxzZQogIHVzZXJuYW1lID0gIjRhNDM3ZjQ2LWRhN2ItNDQ2OS05NmI3LWJlNzU0YjVlOGQzNiIKCltbcm91dGVzXV0KICBtcXR0X3RvcGljID0gIjRjNjZhNzg1LTE5MDAtNDg0NC04Y2FhLTU2ZmI4Y2ZkNjFlYiIKICBuYXRzX3RvcGljID0gIioiCg==\"}]"
```

`vs="config_file_path, file_cont_base64"` - vs determines where to save file and contains file content in base64 encoding payload:
```
b,_ := toml.Marshal(export.Config)
payload := base64.StdEncoding.EncodeToString(b)
```

[conftoml]: (https://github.com/mainflux/export/blob/master/configs/config.toml)