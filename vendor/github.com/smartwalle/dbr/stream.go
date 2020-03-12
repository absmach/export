package dbr

// XACK XACK key group ID [ID ...]
func (this *Session) XACK(key, group string, ids ...interface{}) *Result {
	var ks = make([]interface{}, 0, len(ids)+2)
	ks = append(ks, key, group)
	ks = append(ks, ids...)
	return this.Do("XACK", ks...)
}

// XADD XADD key ID field string [field string ...]
func (this *Session) XADD(key string, maxLen int, id string, args ...interface{}) *Result {
	var ks = make([]interface{}, 0, len(args)+4)
	ks = append(ks, key)
	if maxLen > 0 {
		ks = append(ks, "MAXLEN", maxLen)
	}
	ks = append(ks, id)
	ks = append(ks, args...)
	return this.Do("XADD", ks...)
}

// XDEL XDEL key ID [ID ...]
func (this *Session) XDEL(key string, args ...interface{}) *Result {
	var ks = make([]interface{}, 0, len(args)+1)
	ks = append(ks, key)
	ks = append(ks, args...)
	return this.Do("XDEL", ks...)
}

func (this *Session) XGROUP(command, key, group string, args ...interface{}) *Result {
	var ks = make([]interface{}, 0, len(args)+3)
	ks = append(ks, command, key, group)
	ks = append(ks, args...)
	return this.Do("XGROUP", ks...)
}

// XGROUPCREATE CREATE key groupname id-or-$
// 0-0  表示从头开始消费
// $    表示从尾部开始消费，只接受新消息，当前Stream消息会全部忽略
func (this *Session) XGROUPCREATE(key, group string, args ...interface{}) *Result {
	return this.XGROUP("CREATE", key, group, args...)
}

// XGROUPSETID SETID key groupname id-or-$
func (this *Session) XGROUPSETID(key, group string, args ...interface{}) *Result {
	return this.XGROUP("SETID", key, group, args...)
}

// XGROUPDESTROY DESTROY key groupname
func (this *Session) XGROUPDESTROY(key, group string) *Result {
	return this.XGROUP("DESTROY", key, group)
}

// XGROUPDELCONSUMER DELCONSUMER key groupname consumername
func (this *Session) XGROUPDELCONSUMER(key, group, consumer string) *Result {
	return this.XGROUP("DELCONSUMER", key, group, consumer)
}

// XLEN XLEN key
func (this *Session) XLEN(key string) *Result {
	return this.Do("XLEN", key)
}

// XPENDING XPENDING key group [start end count] [consumer]
func (this *Session) XPENDING(key, group string, args ...interface{}) *Result {
	var ks = make([]interface{}, 0, len(args)+2)
	ks = append(ks, key, group)
	ks = append(ks, args...)
	return this.Do("XPENDING", ks...)
}

// XREAD XREAD [COUNT count] [BLOCK milliseconds] STREAMS key [key ...] ID [ID ...]
func (this *Session) XREAD(count int, milliseconds int64, keysAndIds ...interface{}) *Result {
	var ks = make([]interface{}, 0, len(keysAndIds)+5)
	ks = append(ks, "COUNT", count)
	ks = append(ks, "BLOCK", milliseconds)
	ks = append(ks, "STREAMS")
	ks = append(ks, keysAndIds...)
	return this.Do("XREAD", ks...)
}

// XREADGROUP XREADGROUP GROUP group consumer [COUNT count] [BLOCK milliseconds] [NOACK] STREAMS key [key ...] ID [ID ...]
// >    表示从当前消费组的last_delivered_id后面开始读
// 0-0  表示读取所有的PEL消息以及自last_delivered_id之后的新消息
func (this *Session) XREADGROUP(group, consumer string, count int, milliseconds int64, keysAndIds ...interface{}) *Result {
	var ks = make([]interface{}, 0, len(keysAndIds)+8)
	ks = append(ks, "GROUP", group, consumer)
	ks = append(ks, "COUNT", count)
	ks = append(ks, "BLOCK", milliseconds)
	ks = append(ks, "STREAMS")
	ks = append(ks, keysAndIds...)
	return this.Do("XREADGROUP", ks...)
}
