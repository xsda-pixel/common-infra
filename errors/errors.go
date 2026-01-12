package errors

import "fmt"

type Msg struct {
	Str  string
	Args []any
}

func NewMsg(str string, args ...any) Msg {
	return Msg{Str: str, Args: args}
}

func NewError(code int, msg Msg) Error {
	return &err{Code: code, Msg: msg}
}

func (m Msg) String() string {
	if len(m.Args) == 0 {
		return m.Str
	}
	return fmt.Sprintf(m.Str, m.Args...)
}

type Error interface {
	i()
	ErrCode() int
	ErrMsg() Msg
	Error() string
}

type err struct {
	Code int
	Msg  Msg
}

func (e *err) i() {}

func (e *err) ErrCode() int {
	return e.Code
}

func (e *err) ErrMsg() Msg {
	return e.Msg
}

func (e *err) Error() string {
	return e.Msg.String()
}
