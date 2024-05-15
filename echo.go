package goo

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/ziflex/lecho/v3"
)

type EchoConfig struct {
	Listen string
}

func NewEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	return e
}

func getCustomHTTPErrorHandler(logger *zerolog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		logger.Debug().
			Stringer("url", c.Request().URL).
			Err(err).
			Msg("HTTP error")

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

func ProvideEcho(logger *zerolog.Logger) *echo.Echo {
	e := NewEcho()

	echolog := logger.With().Str("_type", "Echo").Logger()

	e.Logger = lecho.From(echolog)
	e.HTTPErrorHandler = getCustomHTTPErrorHandler(&echolog)

	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	return e
}
