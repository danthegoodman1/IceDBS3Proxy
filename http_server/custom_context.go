package http_server

import (
	"context"
	"errors"
	"github.com/danthegoodman1/GoAPITemplate/utils"
	"net/http"
	"net/url"
	"strings"

	"github.com/danthegoodman1/GoAPITemplate/gologger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type CustomContext struct {
	echo.Context
	RequestID, UserID, VirtualBucketName, RealBucketName string
	IsPathRouting                                        bool
}

func CreateReqContext(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqID := uuid.NewString()
		ctx := context.WithValue(c.Request().Context(), gologger.ReqIDKey, reqID)
		ctx = logger.WithContext(ctx)
		c.SetRequest(c.Request().WithContext(ctx))
		logger := zerolog.Ctx(ctx)
		logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("reqID", reqID)
		})
		cc := &CustomContext{
			Context:   c,
			RequestID: reqID,
		}
		return next(cc)
	}
}

// Casts to custom context for the handler, so this doesn't have to be done per handler
func (srv *HTTPServer) ccHandler(h func(*CustomContext) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*CustomContext)
		domainParts := strings.Split(c.Request().Host, ".")
		if len(domainParts) > len(utils.MyURLParts) {
			// vhost routing
			cc.VirtualBucketName = domainParts[0]
		} else {
			// path routing, path style list possibly
			u, err := url.Parse(c.Request().RequestURI)
			if err != nil {
				return cc.InternalError(err, "error in url.Parse")
			}
			pathParts := strings.Split(u.Path, "/")
			cc.VirtualBucketName = pathParts[1]
			cc.IsPathRouting = true
		}
		return h(cc)
	}
}

func (c *CustomContext) internalErrorMessage() string {
	return "internal error, request id: " + c.RequestID
}

func (c *CustomContext) InternalError(err error, msg string) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		zerolog.Ctx(c.Request().Context()).Warn().CallerSkipFrame(1).Msg(err.Error())
	} else {
		zerolog.Ctx(c.Request().Context()).Error().CallerSkipFrame(1).Err(err).Msg(msg)
	}
	return c.String(http.StatusInternalServerError, c.internalErrorMessage())
}
