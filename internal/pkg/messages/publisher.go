package messages

type Publisher interface {
	Publish(string, []byte) error
}
