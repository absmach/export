# Export
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

```
cd build
./mainflux-export
```

### Config
By default it will look for config file at `../configs/config.toml` if no env vars are specified.  

```
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

[[routes]]
  mqtt_topic = "channel/4c66a785-1900-4844-8caa-56fb8cfd61eb/messages"
  nats_topic = "*"
  type = "mfx"

[[routes]]
  mqtt_topic = "channel/4c66a785-1900-4844-8caa-56fb8cfd61eb/messages"
  nats_topic = "*"
  type = "plain"
```
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
  
## Environmet variables

Service will look for `config.toml` first and if not found it will be configured   
with env variables and new `config.toml` will be saved with values populated from env vars.  
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

## How to save config via agent

```
mosquitto_pub -u <thing_id> -P <thing_key> -t channels/<control_ch_id>/messages/req -h localhost -p 18831  -m  "[{\"bn\":\"1:\", \"n\":\"config\", \"vs\":\"config.toml, RmlsZSA9ICIuLi9jb25maWdzL2NvbmZpZy50b21sIgoKW2V4cF0KICBsb2dfbGV2ZWwgPSAiZGVidWciCiAgbmF0cyA9ICJuYXRzOi8vMTI3LjAuMC4xOjQyMjIiCiAgcG9ydCA9ICI4MTcwIgoKW21xdHRdCiAgY2FfcGF0aCA9ICJjYS5jcnQiCiAgY2VydF9wYXRoID0gInRoaW5nLmNydCIKICBjaGFubmVsID0gIiIKICBob3N0ID0gInRjcDovL2xvY2FsaG9zdDoxODgzIgogIG10bHMgPSBmYWxzZQogIHBhc3N3b3JkID0gImFjNmI1N2UwLTliNzAtNDVkNi05NGM4LWU2N2FjOTA4NjE2NSIKICBwcml2X2tleV9wYXRoID0gInRoaW5nLmtleSIKICBxb3MgPSAwCiAgcmV0YWluID0gZmFsc2UKICBza2lwX3Rsc192ZXIgPSBmYWxzZQogIHVzZXJuYW1lID0gIjRhNDM3ZjQ2LWRhN2ItNDQ2OS05NmI3LWJlNzU0YjVlOGQzNiIKCltbcm91dGVzXV0KICBtcXR0X3RvcGljID0gIjRjNjZhNzg1LTE5MDAtNDg0NC04Y2FhLTU2ZmI4Y2ZkNjFlYiIKICBuYXRzX3RvcGljID0gIioiCg==\"}]"
```

`vs="config_file_path, file_cont_base64"` - vs determines where to save file and contains file content in base64 encoding payload:
```
b,_ := toml.Marshal(export.Config)
payload := base64.StdEncoding.EncodeToString(b)
```