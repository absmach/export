package dbr

import "encoding/json"

////////////////////////////////////////////////////////////////////////////////
// 把一个对象编码成 JSON 字符串数据进行存储
func (this *Session) MarshalJSON(key string, obj interface{}) *Result {
	value, err := json.Marshal(obj)
	if err != nil {
		return result(nil, err)
	}
	return this.SET(key, string(value))
}

func (this *Session) MarshalJSONEx(key string, seconds int64, obj interface{}) *Result {
	value, err := json.Marshal(obj)
	if err != nil {
		return result(nil, err)
	}
	return this.SETEX(key, seconds, string(value))
}

func (this *Session) UnmarshalJSON(key string, des interface{}) (err error) {
	if err = this.GET(key).UnmarshalJSON(des); err != nil {
		return err
	}
	return nil
}

func (this *Result) UnmarshalJSON(des interface{}) (err error) {
	bs, err := this.Bytes()
	if err != nil {
		return err
	}
	if err = json.Unmarshal(bs, des); err != nil {
		return err
	}
	return err
}
