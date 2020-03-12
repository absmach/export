package dbr

//SADD 将一个或多个 member 元素加入到集合 key 当中，已经存在于集合的 member 元素将被忽略。
func (this *Session) SADD(key string, members ...interface{}) *Result {
	var ks []interface{}
	ks = append(ks, key)
	if len(members) > 0 {
		ks = append(ks, members...)
	}
	return this.Do("SADD", ks...)
}

//SCARD 返回集合 key 的基数(集合中元素的数量)。
func (this *Session) SCARD(key string) *Result {
	return this.Do("SCARD", key)
}

//SDIFF 返回一个集合的全部成员，该集合是所有给定集合之间的差集。
func (this *Session) SDIFF(keys ...string) *Result {
	var ks []interface{}
	for _, key := range keys {
		ks = append(ks, key)
	}
	return this.Do("SDIFF", ks...)
}

//SDIFFSTORE 这个命令的作用和 SDIFF 类似，但它将结果保存到 destination 集合，而不是简单地返回结果集。
func (this *Session) SDIFFSTORE(destination string, keys ...string) *Result {
	var ks []interface{}
	ks = append(ks, destination)
	for _, key := range keys {
		ks = append(ks, key)
	}
	return this.Do("SDIFFSTORE", ks...)
}

//SINTER 返回一个集合的全部成员，该集合是所有给定集合的交集。
func (this *Session) SINTER(keys ...string) *Result {
	var ks []interface{}
	for _, key := range keys {
		ks = append(ks, key)
	}
	return this.Do("SINTER", ks...)
}

//SINTERSTORE 这个命令类似于 SINTER 命令，但它将结果保存到 destination 集合，而不是简单地返回结果集。
func (this *Session) SINTERSTORE(destination string, keys ...string) *Result {
	var ks []interface{}
	ks = append(ks, destination)
	for _, key := range keys {
		ks = append(ks, key)
	}
	return this.Do("SINTERSTORE", ks...)
}

//SISMEMBER 判断 member 元素是否集合 key 的成员。
func (this *Session) SISMEMBER(key string, member interface{}) *Result {
	return this.Do("SISMEMBER", key, member)
}

//SMEMBERS 返回集合 key 中的所有成员。
func (this *Session) SMEMBERS(key string) *Result {
	return this.Do("SMEMBERS", key)
}

//SMOVE 将 member 元素从 source 集合移动到 destination 集合。
func (this *Session) SMOVE(source, destination string, member interface{}) *Result {
	return this.Do("SMOVE", source, destination, member)
}

//SPOP 移除并返回集合中的一个随机元素。
func (this *Session) SPOP(key string) *Result {
	return this.Do("SPOP", key)
}

//SRANDMEMBER 如果命令执行时，只提供了 key 参数，那么返回集合中的一个随机元素。
func (this *Session) SRANDMEMBER(key string, count int) *Result {
	return this.Do("SRANDMEMBER", key, count)
}

//SREM 移除集合 key 中的一个或多个 member 元素，不存在的 member 元素会被忽略。
func (this *Session) SREM(key string, members ...interface{}) *Result {
	var ks []interface{}
	ks = append(ks, key)
	if len(members) > 0 {
		ks = append(ks, members...)
	}

	return this.Do("SREM", ks...)
}

//SUNION 返回一个集合的全部成员，该集合是所有给定集合的并集。
func (this *Session) SUNION(keys ...string) *Result {
	var ks []interface{}
	for _, key := range keys {
		ks = append(ks, key)
	}
	return this.Do("SUNION", ks...)
}

//SUNIONSTORE 这个命令类似于 SUNION 命令，但它将结果保存到 destination 集合，而不是简单地返回结果集。
func (this *Session) SUNIONSTORE(destination string, keys ...string) *Result {
	var ks []interface{}
	ks = append(ks, destination)
	for _, key := range keys {
		ks = append(ks, key)
	}
	return this.Do("SUNIONSTORE", ks...)
}
