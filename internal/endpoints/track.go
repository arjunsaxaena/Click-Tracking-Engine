package endpoints

import (
	"context"
	"time"

	"project/internal/service"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/log"
)

type TrackRequest struct {
	LinkID    string
	UserID    string
	GAID      string
	IDFA      string
	IP        string
	UserAgent string
	Referrer  string
}

type TrackResponse struct {
	StatusCode  int
	Body        string
	RedirectURL string
}

type TrackEndpointSet struct {
	TrackEndpoint endpoint.Endpoint
}

func MakeTrackEndpoint(s service.ClickService, logger log.Logger) endpoint.Endpoint {
	ep := func(ctx context.Context, request any) (any, error) {
		req := request.(TrackRequest)

		out, err := s.HandleClick(ctx, service.TrackInput(req))
		if err != nil {
			return nil, err
		}

		return TrackResponse{
			StatusCode:  out.StatusCode,
			Body:        out.Body,
			RedirectURL: out.RedirectURL,
		}, nil
	}

	return LoggingMiddleware(logger)(ep)
}

func LoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request any) (any, error) {
			req := request.(TrackRequest)
			start := time.Now()

			response, err := next(ctx, request)
			duration := time.Since(start)

			if err != nil {
				logger.Log(
					"method", "track",
					"link_id", req.LinkID,
					"user_id", req.UserID,
					"error", err.Error(),
					"duration_ms", duration.Milliseconds(),
					"msg", "request failed",
				)
				return nil, err
			}

			resp := response.(TrackResponse)
			logger.Log(
				"method", "track",
				"link_id", req.LinkID,
				"user_id", req.UserID,
				"status_code", resp.StatusCode,
				"duration_ms", duration.Milliseconds(),
				"msg", "request completed",
			)

			return response, nil
		}
	}
}