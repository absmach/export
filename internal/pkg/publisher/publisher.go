package publisher

import (
	"encoding/json"
	"time"

	"github.com/cloudflare/buffer"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	filename = "store.dat"
	filesize = 262144000
)

type record struct {
	time  time.Time
	topic string
	bytes []byte
}

type storage struct {
	buff *buffer.Buffer
	mqtt mqtt.Client
}
type File interface {
	Publish(b []byte, topic string) (err error)
}

func NewPublisher(name string, m mqtt.Client) (File, error) {
	s := storage{}
	b, err := buffer.New(name, filesize)
	if err != nil {
		return &s, err
	}
	s.buff = b
	s.mqtt = m
	return &s, nil
}
func (s *storage) write(b []byte, topic string) (err error) {
	time := time.Now()
	r := record{time, topic, b}
	w, err := json.Marshal(r)
	if err != nil {
		return err
	}
	err = s.buff.Insert(w)
	return err
}

func (s *storage) Publish(b []byte, topic string) (err error) {
	return nil
}

func (s *storage) read() (b []byte, err error) {
	b, err = s.buff.Pop()
	if err != nil {
		return nil, err
	}
	r := record{}
	err = json.Unmarshal(b, r)
	if err != nil {
		return nil, err
	}
	return r.bytes, nil
}
