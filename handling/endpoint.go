package handling

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/zipkin"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/location"
	"github.com/Qalifah/shipping/voyage"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
)

type registerEventRequest struct {
	ID		cargo.TrackingID
	Location	location.UNLcode
	Voyage		voyage.Number
	EventType	cargo.HandlingEventType
	CompletionTime	time.Time
}

type registerEventResponse struct {
	Err		error	`json:"error,omitempty"`
}

func (r registerEventResponse) error() error {return r.Err }

func makeRegisterEventEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(registerEventRequest)
		err := s.RegisterHandlingEvent(req.CompletionTime, req.ID, req.Voyage, req.Location, req.EventType)
		return registerEventResponse{Err : err}, nil
	}
}

// Set collects all of the endpoints that compose a handling cargo service.
type Set struct {
	RegisterEventEndpoint		endpoint.Endpoint
}

// NewSet returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func NewSet(svc Service, logger log.Logger, duration metrics.Histogram, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer) Set {
	var registerEventEndpoint endpoint.Endpoint
	{
		registerEventEndpoint = makeRegisterEventEndpoint(svc)
		registerEventEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(registerEventEndpoint)
		registerEventEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(registerEventEndpoint)
		if zipkinTracer != nil {
			registerEventEndpoint = zipkin.TraceEndpoint(zipkinTracer, "RequestEvent")(registerEventEndpoint)
		}
	}
	return Set{
		RegisterEventEndpoint: registerEventEndpoint,
	}
}

// RegisterHandlingEvent implements the service interface so Set can be used as a service
func(s Set) RegisterHandlingEvent(completed time.Time, id cargo.TrackingID, voyageNumber voyage.Number, unLcode location.UNLcode, eventType cargo.HandlingEventType) error {
	resp, err := s.RegisterEventEndpoint(context.Background(), registerEventRequest{ID: id, Location: unLcode, Voyage: voyageNumber, EventType: eventType, CompletionTime: completed})
	if err != nil {
		return err
	}
	response := resp.(registerEventResponse)
	return response.Err
}