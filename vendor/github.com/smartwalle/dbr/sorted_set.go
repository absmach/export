package dbr

// ZADD 将一个或多个 member 元素及其 score 值加入到有序集 key 当中。
// ZADD key score member [[score member] [score member] ...]
func (this *Session) ZADD(key string, params ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	if len(params) > 0 {
		ps = append(ps, params...)
	}
	return this.Do("ZADD", ps...)
}

// ZCARD 返回有序集 key 的基数(集合中元素的数量)。
func (this *Session) ZCARD(key string) *Result {
	return this.Do("ZCARD", key)
}

//ZCOUNT 返回有序集 key 中， score 值在 min 和 max 之间(默认包括 score 值等于 min 或 max )的成员的数量。
func (this *Session) ZCOUNT(key string, min, max float64) *Result {
	return this.Do("ZCOUNT", key, min, max)
}

//ZINCRBY 为有序集 key 的成员 member 的 score 值加上增量 increment 。
func (this *Session) ZINCRBY(key string, increment float64, member string) *Result {
	return this.Do("ZINCRBY", key, increment, member)
}

//ZRANGE 返回有序集 key 中，指定区间内的成员。
func (this *Session) ZRANGE(key string, start, stop int, options ...string) *Result {
	var ps []interface{}
	ps = append(ps, key)
	ps = append(ps, start)
	ps = append(ps, stop)
	for _, o := range options {
		ps = append(ps, o)
	}
	return this.Do("ZRANGE", ps...)
}

// ZRANGEBYSCORE 返回有序集 key 中，所有 score 值介于 min 和 max 之间(包括等于 min 或 max )的成员。有序集成员按 score 值递增(从小到大)次序排列。
func (this *Session) ZRANGEBYSCORE(key string, min, max float64, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	ps = append(ps, min)
	ps = append(ps, max)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("ZRANGEBYSCORE", ps...)
}

//ZRANK 返回有序集 key 中成员 member 的排名。其中有序集成员按 score 值递增(从小到大)顺序排列。
func (this *Session) ZRANK(key, member string) *Result {
	return this.Do("ZRANK", key, member)
}

//ZREM 移除有序集 key 中的一个或多个成员，不存在的成员将被忽略。
func (this *Session) ZREM(key string, members ...string) *Result {
	var ps []interface{}
	ps = append(ps, key)
	for _, m := range members {
		ps = append(ps, m)
	}
	return this.Do("ZREM", ps...)
}

//ZREMRANGEBYRANK 移除有序集 key 中，指定排名(rank)区间内的所有成员。
func (this *Session) ZREMRANGEBYRANK(key string, start, stop int) *Result {
	return this.Do("ZREMRANGEBYRANK", key, start, stop)
}

//ZREMRANGEBYSCORE 移除有序集 key 中，所有 score 值介于 min 和 max 之间(包括等于 min 或 max )的成员。
func (this *Session) ZREMRANGEBYSCORE(key string, min, max float64) *Result {
	return this.Do("ZREMRANGEBYSCORE", key, min, max)
}

//ZREVRANGE 返回有序集 key 中，指定区间内的成员。
func (this *Session) ZREVRANGE(key string, start, stop int, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	ps = append(ps, start)
	ps = append(ps, stop)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("ZREVRANGE", ps...)
}

//ZREVRANGEBYSCORE 返回有序集 key 中， score 值介于 max 和 min 之间(默认包括等于 max 或 min )的所有的成员。有序集成员按 score 值递减(从大到小)的次序排列。
func (this *Session) ZREVRANGEBYSCORE(key string, max, min float64, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	ps = append(ps, max)
	ps = append(ps, min)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("ZREVRANGEBYSCORE", ps...)
}

//ZREVRANK 返回有序集 key 中成员 member 的排名。其中有序集成员按 score 值递减(从大到小)排序。
func (this *Session) ZREVRANK(key, member string) *Result {
	return this.Do("ZREVRANK", key, member)
}

//ZSCORE 返回有序集 key 中，成员 member 的 score 值。
func (this *Session) ZSCORE(key, member string) *Result {
	return this.Do("ZSCORE", key, member)
}

//ZUNIONSTORE 计算给定的一个或多个有序集的并集，其中给定 key 的数量必须以 numkeys 参数指定，并将该并集(结果集)储存到 destination 。
func (this *Session) ZUNIONSTORE(destination string, numKeys int, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, destination)
	ps = append(ps, numKeys)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("ZUNIONSTORE", ps...)
}

//ZINTERSTORE 计算给定的一个或多个有序集的交集，其中给定 key 的数量必须以 numkeys 参数指定，并将该交集(结果集)储存到 destination 。
func (this *Session) ZINTERSTORE(destination string, numKeys int, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, destination)
	ps = append(ps, numKeys)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("ZINTERSTORE", ps...)
}

//RANGEBYLEX 当有序集合的所有成员都具有相同的分值时， 有序集合的元素会根据成员的字典序（lexicographical ordering）来进行排序， 而这个命令则可以返回给定的有序集合键 key 中， 值介于 min 和 max 之间的成员。
func (this *Session) RANGEBYLEX(key string, min, max float64, options ...interface{}) *Result {
	var ps []interface{}
	ps = append(ps, key)
	ps = append(ps, min)
	ps = append(ps, max)
	if len(options) > 0 {
		ps = append(ps, options...)
	}
	return this.Do("RANGEBYLEX", ps...)
}

//ZLEXCOUNT 对于一个所有成员的分值都相同的有序集合键 key 来说， 这个命令会返回该集合中， 成员介于 min 和 max 范围内的元素数量。
func (this *Session) ZLEXCOUNT(key string, min, max float64) *Result {
	return this.Do("ZLEXCOUNT", key, min, max)
}

//ZREMRANGEBYLEX 对于一个所有成员的分值都相同的有序集合键 key 来说， 这个命令会移除该集合中， 成员介于 min 和 max 范围内的所有元素。
func (this *Session) ZREMRANGEBYLEX(key string, min, max float64) *Result {
	return this.Do("ZREMRANGEBYLEX", key, min, max)
}
