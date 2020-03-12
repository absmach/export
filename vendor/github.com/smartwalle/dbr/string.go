package dbr

//APPEND 如果 key 已经存在并且是一个字符串， APPEND 命令将 value 追加到 key 原来的值的末尾。
func (this *Session) APPEND(key, value string) *Result {
	return this.Do("APPEND", key, value)
}

//DECR 将 key 中储存的数字值减一。
func (this *Session) DECR(key string) *Result {
	return this.Do("DECR", key)
}

//DECRBY 将 key 所储存的值减去减量 decrement 。
func (this *Session) DECRBY(key string, decrement int) *Result {
	return this.Do("DECRBY", key, decrement)
}

//GET 返回 key 所关联的字符串值。
func (this *Session) GET(key string) *Result {
	return this.Do("GET", key)
}

//GETBIT 对 key 所储存的字符串值，获取指定偏移量上的位(bit)。
func (this *Session) GETBIT(key string, offset int) *Result {
	return this.Do("GETBIT", key, offset)
}

//GETRANGE 返回 key 中字符串值的子字符串，字符串的截取范围由 start 和 end 两个偏移量决定(包括 start 和 end 在内)。
func (this *Session) GETRANGE(key string, start, end int) *Result {
	return this.Do("GETRANGE", key, start, end)
}

//GETSET 将给定 key 的值设为 value ，并返回 key 的旧值(old value)。
func (this *Session) GETSET(key string, value interface{}) *Result {
	return this.Do("GETSET", key, value)
}

//INCR 将 key 中储存的数字值增一。
func (this *Session) INCR(key string) *Result {
	return this.Do("INCR", key)
}

//INCRBY 将 key 所储存的值加上增量 increment 。
func (this *Session) INCRBY(key string, increment int) *Result {
	return this.Do("INCRBY", key, increment)
}

//INCRBYFLOAT 为 key 中所储存的值加上浮点数增量 increment 。
func (this *Session) INCRBYFLOAT(key string, increment float64) *Result {
	return this.Do("INCRBYFLOAT", key, increment)
}

//MGET 返回所有(一个或多个)给定 key 的值。
func (this *Session) MGET(keys ...string) *Result {
	var ks []interface{}
	for _, k := range keys {
		ks = append(ks, k)
	}
	return this.Do("MGET", ks...)
}

//MSET 同时设置一个或多个 key-value 对。
func (this *Session) MSET(params ...interface{}) *Result {
	return this.Do("MSET", params...)
}

//MSETNX 同时设置一个或多个 key-value 对，当且仅当所有给定 key 都不存在。 即使只有一个给定 key 已存在， MSETNX 也会拒绝执行所有给定 key 的设置操作。
func (this *Session) MSETNX(params ...interface{}) *Result {
	return this.Do("MSETNX", params...)
}

//PSETEX 这个命令和 SETEX 命令相似，但它以毫秒为单位设置 key 的生存时间，而不是像 SETEX 命令那样，以秒为单位。
func (this *Session) PSETEX(key string, milliseconds int64, value interface{}) *Result {
	return this.Do("PSETEX", key, milliseconds, value)
}

//SET 将字符串值 value 关联到 key 。
//
//SET key value [EX seconds] [PX milliseconds] [NX|XX]
//如果 key 已经持有其他值， SET 就覆写旧值，无视类型。
//对于某个原本带有生存时间（TTL）的键来说， 当 SET 命令成功在这个键上执行时， 这个键原有的 TTL 将被清除。
//
//可选参数
//从 Redis 2.6.12 版本开始， SET 命令的行为可以通过一系列参数来修改：
//EX second ：设置键的过期时间为 second 秒。 SET key value EX second 效果等同于 SETEX key second value 。
//PX millisecond ：设置键的过期时间为 millisecond 毫秒。 SET key value PX millisecond 效果等同于 PSETEX key millisecond value 。
//NX ：只在键不存在时，才对键进行设置操作。 SET key value NX 效果等同于 SETNX key value 。
//XX ：只在键已经存在时，才对键进行设置操作。
func (this *Session) SET(params ...interface{}) *Result {
	return this.Do("SET", params...)
}

//SETBIT 对 key 所储存的字符串值，设置或清除指定偏移量上的位(bit)。
func (this *Session) SETBIT(key string, offset int, value string) *Result {
	return this.Do("SETBIT", key, offset, value)
}

//SETEX 将值 value 关联到 key ，并将 key 的生存时间设为 seconds (以秒为单位)。
func (this *Session) SETEX(key string, seconds int64, value interface{}) *Result {
	return this.Do("SETEX", key, seconds, value)
}

//SETNX 将 key 的值设为 value ，当且仅当 key 不存在。
func (this *Session) SETNX(key string, value interface{}) *Result {
	return this.Do("SETNX", key, value)
}

//SETRANGE 用 value 参数覆写(overwrite)给定 key 所储存的字符串值，从偏移量 offset 开始。
func (this *Session) SETRANGE(key string, offset int, value string) *Result {
	return this.Do("SETRANGE", key, offset, value)
}

//STRLEN 返回 key 所储存的字符串值的长度。
func (this *Session) STRLEN(key string) *Result {
	return this.Do("STRLEN", key)
}
