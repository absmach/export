package dbr

//DEL 删除给定的一个或多个 key 。
func (this *Session) DEL(keys ...string) *Result {
	var ks []interface{}
	for _, k := range keys {
		ks = append(ks, k)
	}
	return this.Do("DEL", ks...)
}

//DUMP 序列化给定 key ，并返回被序列化的值，使用 RESTORE 命令可以将这个值反序列化为 Redis 键。
func (this *Session) DUMP(key string) *Result {
	return this.Do("DUMP", key)
}

//EXISTS 检查给定 key 是否存在。
func (this *Session) EXISTS(key string) *Result {
	return this.Do("EXISTS", key)
}

//EXPIRE 为给定 key 设置生存时间，当 key 过期时(生存时间为 0 )，它会被自动删除。
func (this *Session) EXPIRE(key string, seconds int64) *Result {
	return this.Do("EXPIRE", key, seconds)
}

//EXPIREAT 作用和 EXPIRE 类似，都用于为 key 设置生存时间。不同在于 EXPIREAT 命令接受的时间参数是 UNIX 时间戳(unix timestamp)。
func (this *Session) EXPIREAT(key string, timestamp int64) *Result {
	return this.Do("EXPIREAT", key, timestamp)
}

//KEYS 查找所有符合给定模式 pattern 的 key 。
func (this *Session) KEYS(pattern string) *Result {
	return this.Do("KEYS", pattern)
}

//MIGRATE 将 key 原子性地从当前实例传送到目标实例的指定数据库上，一旦传送成功， key 保证会出现在目标实例上，而当前实例上的 key 会被删除。
func (this *Session) MIGRATE(host, port, key string, destinationDB int, timeout int, options ...string) *Result {
	var ps []interface{}
	ps = append(ps, host)
	ps = append(ps, port)
	ps = append(ps, key)
	ps = append(ps, destinationDB)
	ps = append(ps, timeout)
	for _, o := range options {
		ps = append(ps, o)
	}
	return this.Do("MIGRATE", ps...)
}

//MOVE 将当前数据库的 key 移动到给定的数据库 db 当中。
func (this *Session) MOVE(key string, destinationDB int) *Result {
	return this.Do("MOVE", key, destinationDB)
}

//PERSIST 移除给定 key 的生存时间，将这个 key 从『易失的』(带生存时间 key )转换成『持久的』(一个不带生存时间、永不过期的 key)。
func (this *Session) PERSIST(key string) *Result {
	return this.Do("PERSIST", key)
}

//PEXPIRE 这个命令和 EXPIRE 命令的作用类似，但是它以毫秒为单位设置 key 的生存时间，而不像 EXPIRE 命令那样，以秒为单位。
func (this *Session) PEXPIRE(key string, milliseconds int64) *Result {
	return this.Do("PEXPIRE", key, milliseconds)
}

//PEXPIREAT 这个命令和 EXPIREAT 命令类似，但它以毫秒为单位设置 key 的过期 unix 时间戳，而不是像 EXPIREAT 那样，以秒为单位。
func (this *Session) PEXPIREAT(key string, timestamp int64) *Result {
	return this.Do("PEXPIREAT", key, timestamp)
}

//PTTL 这个命令类似于 TTL 命令，但它以毫秒为单位返回 key 的剩余生存时间，而不是像 TTL 命令那样，以秒为单位。
func (this *Session) PTTL(key string) *Result {
	return this.Do("PTTL", key)
}

//RANDOMKEY 从当前数据库中随机返回(不删除)一个 key
func (this *Session) RANDOMKEY() *Result {
	return this.Do("RANDOMKEY")
}

//RENAME 将 key 改名为 newKey。
func (this *Session) RENAME(key, newKey string) *Result {
	return this.Do("RENAME", key, newKey)
}

//RENAMENX 当且仅当 newkey 不存在时，将 key 改名为 newkey。
func (this *Session) RENAMENX(key, newKey string) *Result {
	return this.Do("RENAMENX", key, newKey)
}

//RESTORE 反序列化给定的序列化值，并将它和给定的 key 关联。
func (this *Session) RESTORE(key string, ttl int64, serializedValue string, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	ps = append(ps, ttl)
	ps = append(ps, serializedValue)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("RESTORE", ps...)
}

//SORT 返回或保存给定列表、集合、有序集合 key 中经过排序的元素。
func (this *Session) SORT(key string, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("SORT", ps...)
}

//TTL 以秒为单位，返回给定 key 的剩余生存时间(TTL, time to live)。
func (this *Session) TTL(key string) *Result {
	return this.Do("TTL", key)
}

//TYPE 返回 key 所储存的值的类型。
func (this *Session) TYPE(key string) *Result {
	return this.Do("TYPE", key)
}
