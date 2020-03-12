package dbr

type GEOUnit string

const (
	GEOUnitM  GEOUnit = "m"  // 表示单位为米。
	GEOUnitKM GEOUnit = "km" // 表示单位为千米。
	GEOUnitMI GEOUnit = "mi" // 表示单位为千米。
	GEOUnitFT GEOUnit = "ft" // 表示单位为英尺
)

func (this *Session) GEOADD(key, longitude, latitude, member string) *Result {
	return this.Do("GEOADD", key, longitude, latitude, member)
}

func (this *Session) GEODIST(key, member1, member2 string, unit GEOUnit) *Result {
	return this.Do("GEODIST", key, member1, member2, unit)
}

func (this *Session) GEOHASH(key string, members ...string) *Result {
	var ps = make([]interface{}, 0, 1+len(members))
	ps = append(ps, key)
	for _, m := range members {
		ps = append(ps, m)
	}
	return this.Do("GEOHASH", ps...)
}

func (this *Session) GEOPOS(key string, members ...string) *Result {
	var ps = make([]interface{}, 0, 1+len(members))
	ps = append(ps, key)
	for _, m := range members {
		ps = append(ps, m)
	}
	return this.Do("GEOPOS", ps...)
}

func (this *Session) GEORADIUS(key, longitude, latitude, radius string, unit GEOUnit, options ...string) *Result {
	var ps = make([]interface{}, 0, 4+len(options))
	ps = append(ps, key)
	ps = append(ps, longitude)
	ps = append(ps, latitude)
	ps = append(ps, radius)
	ps = append(ps, string(unit))
	for _, opt := range options {
		ps = append(ps, opt)
	}
	return this.Do("GEORADIUS", ps...)
}

func (this *Session) GEORADIUSBYMEMBER(key, member, radius string, unit GEOUnit, options ...string) *Result {
	var ps = make([]interface{}, 0, 4+len(options))
	ps = append(ps, key)
	ps = append(ps, member)
	ps = append(ps, radius)
	ps = append(ps, string(unit))
	for _, opt := range options {
		ps = append(ps, opt)
	}
	return this.Do("GEORADIUSBYMEMBER", ps...)
}
