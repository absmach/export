package dbr

func (this *Session) BLPOP(params ...interface{}) *Result {
	return this.Do("BLPOP", params...)
}

func (this *Session) BRPOP(params ...interface{}) *Result {
	return this.Do("BRPOP", params...)
}

func (this *Session) BRPOPLPUSH(source, destination string, timeout int) *Result {
	return this.Do("BRPOPLPUSH", source, destination, timeout)
}

func (this *Session) LINDEX(key string, index int) *Result {
	return this.Do("LINDEX", key, index)
}

func (this *Session) LINSERT(key, option, pivot string, value interface{}) *Result {
	return this.Do("LINSERT", key, option, pivot, value)
}

func (this *Session) LLEN(key string) *Result {
	return this.Do("LLEN", key)
}

func (this *Session) LPOP(key string) *Result {
	return this.Do("LPOP", key)
}

func (this *Session) LPUSH(key string, values ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	if len(values) > 0 {
		ps = append(ps, values...)
	}
	return this.Do("LPUSH", ps...)
}

//LPUSHX 将值 value 插入到列表 key 的表头，当且仅当 key 存在并且是一个列表。
func (this *Session) LPUSHX(key string, value interface{}) *Result {
	return this.Do("LPUSHX", key, value)
}

//LRANGE 返回列表 key 中指定区间内的元素，区间以偏移量 start 和 stop 指定。
func (this *Session) LRANGE(key string, start, stop int) *Result {
	return this.Do("LRANGE", key, start, stop)
}

//LREM 根据参数 count 的值，移除列表中与参数 value 相等的元素。
func (this *Session) LREM(key string, count int, value interface{}) *Result {
	return this.Do("LREM", key, count, value)
}

//LSET 将列表 key 下标为 index 的元素的值设置为 value 。
func (this *Session) LSET(key string, index int, value interface{}) *Result {
	return this.Do("LSET", key, index, value)
}

//LTRIM 对一个列表进行修剪(trim)，就是说，让列表只保留指定区间内的元素，不在指定区间之内的元素都将被删除。
func (this *Session) LTRIM(key string, start, stop int) *Result {
	return this.Do("LTRIM", key, start, stop)
}

//RPOP 移除并返回列表 key 的尾元素。
func (this *Session) RPOP(key string) *Result {
	return this.Do("RPOP", key)
}

//RPOPLPUSH
func (this *Session) RPOPLPUSH(source, destination string) *Result {
	return this.Do("RPOPLPUSH", source, destination)
}

//RPUSH 将一个或多个值 value 插入到列表 key 的表尾(最右边)。
func (this *Session) RPUSH(key string, values ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	if len(values) > 0 {
		ps = append(ps, values...)
	}
	return this.Do("RPUSH", ps...)
}

//RPUSHX 将值 value 插入到列表 key 的表尾，当且仅当 key 存在并且是一个列表。
func (this *Session) RPUSHX(key string, value interface{}) *Result {
	return this.Do("RPUSHX", key, value)
}
