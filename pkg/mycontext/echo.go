package mycontext

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

type EchoContextAdapter struct {
	c echo.Context
}

func NewEchoContextAdapter(c echo.Context) *EchoContextAdapter {
	return &EchoContextAdapter{c: c}
}

func (a *EchoContextAdapter) Deadline() (deadline time.Time, ok bool) {
	return a.c.Request().Context().Deadline()
}

func (a *EchoContextAdapter) Done() <-chan struct{} {
	return a.c.Request().Context().Done()
}

func (a *EchoContextAdapter) Err() error {
	return a.c.Request().Context().Err()
}

func (a *EchoContextAdapter) Value(key interface{}) interface{} {
	return a.c.Get(fmt.Sprintf("%v", key))
}
