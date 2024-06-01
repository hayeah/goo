package goo

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
)

type EchoConfig struct {
	Listen string
}

func NewEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	return e
}

func getCustomHTTPErrorHandler(log *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		log.Debug("HTTP error",
			"url", c.Request().URL,
			"error", err)

		code := http.StatusInternalServerError

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}

		c.JSON(code, map[string]interface{}{
			"code":    code,
			"message": err.Error(),
		})
	}
}

func ProvideEcho(baselog *slog.Logger) *echo.Echo {
	e := NewEcho()

	log := baselog.With("_type", "Echo")

	// e.Logger = lecho.From(echolog)
	e.HTTPErrorHandler = getCustomHTTPErrorHandler(log)

	e.Use(slogecho.New(log))

	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	return e
}
