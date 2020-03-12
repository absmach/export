package dbr

func (this *Session) Receive() *Result {
	if this.c != nil {
		return result(this.c.Receive())
	}
	return result(nil, ErrInvalidConn)
}

//var p = dbr.NewRedis("127.0.0.1:6379", "", 1, 30, 1)
//
//var rSess = p.GetSession()
//defer rSess.Close()
//
//rSess.Send(...)
//rSess.Send(...)
//rSess.Send(...)
//rSess.Flush()
//rSess.Receive()
//rSess.Receive()
//rSess.Receive()

func (this *Session) Flush() error {
	if this.c != nil {
		return this.c.Flush()
	}
	return ErrInvalidConn
}
