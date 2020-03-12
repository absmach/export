package dbr

import (
	"errors"
	"github.com/gomodule/redigo/redis"
)

type Result struct {
	Data  interface{}
	Error error
}

func result(data interface{}, err error) *Result {
	return &Result{data, err}
}

////////////////////////////////////////////////////////////////////////////////
func (this *Result) Values() ([]interface{}, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	return Values(this.Data, this.Error)
}

func (this *Result) Bytes() ([]byte, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	return Bytes(this.Data, this.Error)
}

func (this *Result) Int() (int, error) {
	if this.Error != nil {
		return 0, this.Error
	}
	return Int(this.Data, this.Error)
}

func (this *Result) Ints() ([]int, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	return Ints(this.Data, this.Error)
}

func (this *Result) Int64() (int64, error) {
	if this.Error != nil {
		return 0, this.Error
	}
	return Int64(this.Data, this.Error)
}

func (this *Result) Bool() (bool, error) {
	if this.Error != nil {
		return false, this.Error
	}
	return Bool(this.Data, this.Error)
}

func (this *Result) String() (string, error) {
	if this.Error != nil {
		return "", this.Error
	}
	return String(this.Data, this.Error)
}

func (this *Result) Strings() ([]string, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	return Strings(this.Data, this.Error)
}

func (this *Result) Float64() (float64, error) {
	if this.Error != nil {
		return 0.0, this.Error
	}
	return Float64(this.Data, this.Error)
}

func (this *Result) Map() (map[string]string, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	return Map(this.Data, this.Error)
}

////////////////////////////////////////////////////////////////////////////////
func (this *Result) Streams() ([]*StreamInfo, error) {
	if this.Error != nil {
		return nil, this.Error
	}
	return Streams(this.Data, this.Error)
}

////////////////////////////////////////////////////////////////////////////////
func (this *Result) MustValues() []interface{} {
	var r, _ = this.Values()
	return r
}

func (this *Result) MustBytes() []byte {
	var r, _ = this.Bytes()
	return r
}

func (this *Result) MustInt() int {
	var r, _ = this.Int()
	return r
}

func (this *Result) MustInts() []int {
	var r, _ = this.Ints()
	return r
}

func (this *Result) MustInt64() int64 {
	var r, _ = this.Int64()
	return r
}

func (this *Result) MustBool() bool {
	var r, _ = this.Bool()
	return r
}

func (this *Result) MustString() string {
	var r, _ = this.String()
	return r
}

func (this *Result) MustStrings() []string {
	var r, _ = this.Strings()
	return r
}

func (this *Result) MustFloat64() float64 {
	var r, _ = this.Float64()
	return r
}

func (this *Result) MustMap() map[string]string {
	var r, _ = this.Map()
	if r == nil {
		r = make(map[string]string)
	}
	return r
}

////////////////////////////////////////////////////////////////////////////////
func (this *Result) MustStreams() []*StreamInfo {
	var r, _ = this.Streams()
	return r
}

////////////////////////////////////////////////////////////////////////////////
func (this *Result) ScanStruct(destination interface{}) error {
	var err = ScanStruct(this.Data, destination)
	return err
}

////////////////////////////////////////////////////////////////////////////////
func Values(reply interface{}, err error) ([]interface{}, error) {
	return redis.Values(reply, err)
}

func Bytes(reply interface{}, err error) ([]byte, error) {
	return redis.Bytes(reply, err)
}

func Int(reply interface{}, err error) (int, error) {
	return redis.Int(reply, err)
}

func Ints(reply interface{}, err error) ([]int, error) {
	return redis.Ints(reply, err)
}

func Int64(reply interface{}, err error) (int64, error) {
	return redis.Int64(reply, err)
}

func Bool(reply interface{}, err error) (bool, error) {
	return redis.Bool(reply, err)
}

func String(reply interface{}, err error) (string, error) {
	return redis.String(reply, err)
}

func Strings(reply interface{}, err error) ([]string, error) {
	return redis.Strings(reply, err)
}

func Float64(reply interface{}, err error) (float64, error) {
	return redis.Float64(reply, err)
}

func Map(reply interface{}, err error) (map[string]string, error) {
	return redis.StringMap(reply, err)
}

func Streams(reply interface{}, err error) ([]*StreamInfo, error) {
	values, err := redis.Values(reply, err)
	if err != nil {
		return nil, err
	}

	var sList []*StreamInfo
	var kLen = len(values)

	for kIndex := 0; kIndex < kLen; kIndex++ {
		var keyInfo = values[kIndex].([]interface{})

		var key = string(keyInfo[0].([]byte))
		var idList = keyInfo[1].([]interface{})
		var idLen = len(idList)

		sList = make([]*StreamInfo, 0, idLen)

		for idIndex := 0; idIndex < idLen; idIndex++ {
			var idInfo = idList[idIndex].([]interface{})

			var id = string(idInfo[0].([]byte))

			var fieldList = idInfo[1].([]interface{})
			var fLen = len(fieldList)

			if fLen > 0 {
				var stream = &StreamInfo{}
				stream.Key = key
				stream.Id = id
				stream.Fields = make(map[string]string)
				sList = append(sList, stream)

				for fIndex := 0; fIndex < fLen/2; fIndex++ {
					var field = string(fieldList[fIndex*2].([]byte))
					var value = string(fieldList[fIndex*2+1].([]byte))
					stream.Fields[field] = value
				}
			}
		}
	}
	return sList, nil
}

func MustValues(reply interface{}, err error) []interface{} {
	var r, _ = Values(reply, err)
	return r
}

func MustBytes(reply interface{}, err error) []byte {
	var r, _ = Bytes(reply, err)
	return r
}

func MustInt(reply interface{}, err error) int {
	var r, _ = Int(reply, err)
	return r
}

func MustInts(reply interface{}, err error) []int {
	var r, _ = Ints(reply, err)
	return r
}

func MustInt64(reply interface{}, err error) int64 {
	var r, _ = Int64(reply, err)
	return r
}

func MustBool(reply interface{}, err error) bool {
	var r, _ = Bool(reply, err)
	return r
}

func MustString(reply interface{}, err error) string {
	var r, _ = String(reply, err)
	return r
}

func MustStrings(reply interface{}, err error) []string {
	var r, _ = Strings(reply, err)
	return r
}

func MustFloat64(reply interface{}, err error) float64 {
	var r, _ = Float64(reply, err)
	return r
}

func MustMap(reply interface{}, err error) map[string]string {
	var r, _ = Map(reply, err)
	if r == nil {
		r = make(map[string]string)
	}
	return r
}

func MustStreams(reply interface{}, err error) []*StreamInfo {
	var r, _ = Streams(reply, err)
	return r
}

func ScanStruct(source, destination interface{}) error {
	if len(MustValues(source, nil)) == 0 {
		return errors.New("source argument is nil")
	}

	var err = redis.ScanStruct(source.([]interface{}), destination)
	return err
}

func StructToArgs(key string, obj interface{}) redis.Args {
	return redis.Args{}.Add(key).AddFlat(obj)
}

////////////////////////////////////////////////////////////////////////////////
type StreamInfo struct {
	Key    string
	Id     string
	Fields map[string]string
}
