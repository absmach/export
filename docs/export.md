# Export service
Export service is build to bridge between two Mainflux instances. For example we can run Mainflux on the gateway and
we want to export the data to cloud instance of Mainflux as well.
Export service is listening to the NATS and forwards payload to the specified MQTT channel.
 

## Run Export service

Easiest way to run export service is 

```
git clone github.com/mainflux/export
cd github.com/mainflux/export
make
cd build
./mainflux-export
```

service will pickup config file  `github.com/mainflux/export/configs/config.toml`

To configure export service edit config.toml. In order to connect to your Mainflux cloud instance you need to 
provide 
* username - Mainflux thing id
* password - Mainflux thing key
* channel -  should be in format channels/<channel_id>/messages where channel_id is Mainflux channel assigned to thing
* host - MQTT host '''tcp://host.name:1883''' for plain or tcps://host.name:8883 for mtls
  
data will be published to 
`channels/<channel_id>/messages/<orig_channel_id>/<orig_thing_id>/<orig_subtopic>`

`<orig_channel_id>`, `<orig_thing_id>`,`<orig_subtopic>` are extracted from NATS received `mainflux.Message` 
and they are representing thing and channel on gateway instance of Mainflux


you can start export service in docker as well
```docker-compose -f docker/docker-compose.yml up```

this requires that you have previously brought up Mainflux instance with docker-compose 