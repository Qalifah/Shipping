package tracking

import (
	"context"

	"golang.org/x/time/rate"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/zipkin"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
)

type trackCargoRequest struct {
	ID string
}

type trackCargoResponse struct {
	Cargo *Cargo `json:"cargo,omitempty"`
	Err   error  `json:"error,omitempty"`
}

func (r trackCargoResponse) error() error { return r.Err }

func makeTrackCargoEndpoint(ts Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(trackCargoRequest)
		c, err := ts.Track(req.ID)
		return trackCargoResponse{Cargo: &c, Err: err}, nil
	}
}

// Set collects all of the endpoints that compose a handling cargo service.
type Set struct {
	TrackCargoEndpoint endpoint.Endpoint
}

// NewSet returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func NewSet(svc Service, logger log.Logger, duration metrics.Histogram, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer) Set {
	var trackCargoEndpoint endpoint.Endpoint 
	{
		trackCargoEndpoint = makeTrackCargoEndpoint(svc)
		trackCargoEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(trackCargoEndpoint)
		trackCargoEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(trackCargoEndpoint)
		if zipkinTracer != nil {
			trackCargoEndpoint = zipkin.TraceEndpoint(zipkinTracer, "Track Cargo")(trackCargoEndpoint)
		}
	}
	return Set{
		TrackCargoEndpoint: trackCargoEndpoint,
	}
}

// Track implements the service interface so Set can be used as a service
func(s Set) Track(id string) (Cargo, error) {
	resp, err := s.TrackCargoEndpoint(context.Background(), trackCargoRequest{ID: id})
	if err != nil {
		return Cargo{}, err
	}
	response := resp.(trackCargoResponse)
	return *response.Cargo, response.Err
}