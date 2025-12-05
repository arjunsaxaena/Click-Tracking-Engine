package transport

import (
	"context"
	"net"
	"net/http"

	"project/internal/endpoints"

	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
)

func NewHTTPHandler(e endpoints.TrackEndpointSet) http.Handler {
	r := chi.NewRouter()

	r.Method("GET", "/track/{link_id}", kithttp.NewServer(
		e.TrackEndpoint,
		decodeTrackRequest,
		encodeTrackResponse,
	))

	return r
}

func decodeTrackRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return endpoints.TrackRequest{
		LinkID:    chi.URLParam(r, "link_id"),
		UserID:    r.URL.Query().Get("user_id"),
		GAID:      r.URL.Query().Get("gaid"),
		IDFA:      r.URL.Query().Get("idfa"),
		IP:        getIP(r),
		UserAgent: r.UserAgent(),
		Referrer:  r.Referer(),
	}, nil
}

func encodeTrackResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	r := resp.(endpoints.TrackResponse)
	
	if r.StatusCode == 302 && r.RedirectURL != "" {
		w.Header().Set("Location", r.RedirectURL)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(302)
		_, err := w.Write([]byte(r.Body))
		return err
	}
	
	if len(r.Body) > 0 && r.Body[0] == '<' {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	
	w.WriteHeader(r.StatusCode)
	_, err := w.Write([]byte(r.Body))
	return err
}

func getIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return x
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}