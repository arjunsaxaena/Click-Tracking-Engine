package endpoints

import (
	"context"

	"project/internal/service"

	"github.com/go-kit/kit/endpoint"
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

func MakeTrackEndpoint(s service.ClickService) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
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
}