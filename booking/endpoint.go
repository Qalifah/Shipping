package booking

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/location"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/go-kit/kit/tracing/zipkin"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
)

type bookCargoRequest struct {
	Origin	location.UNLcode
	Destination		location.UNLcode
	ArrivalDeadline		time.Time
}

type bookCargoResponse struct {
	ID 		cargo.TrackingID	`json:"tracking_id,omitempty"`
	Err		error				`json:"error,omitempty"`
}

func(r bookCargoResponse) error() error { return r.Err }

func makeBookCargoEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(bookCargoRequest)
		id, err := s.BookNewCargo(req.Origin, req.Destination, req.ArrivalDeadline)
		return bookCargoResponse{ID: id, Err: err}, nil
	}
}

type loadCargoRequest struct {
	ID 	cargo.TrackingID
}

type loadCargoResponse struct {
	Cargo *Cargo	`json:"cargo,omitempty"`
	Err		error	`json:"error,omitempty"`
}

func(r loadCargoResponse) error() error { return r.Err }

func makeLoadCargoEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(loadCargoRequest)
		c, err := s.LoadCargo(req.ID)
		return loadCargoResponse{Cargo: &c, Err: err}, nil
	}
}

type requestRoutesRequest struct {
	ID cargo.TrackingID
}

type requestRoutesResponse struct {
	Routes []cargo.Itinerary	`json:"routes,omitempty"`
	Err		error				`json:"error,omitempty"`
}

func(r requestRoutesResponse) error() error { return r.Err }

func makeRequestRoutesEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(requestRoutesRequest)
		itin := s.RequestPossibleRoutesForCargo(req.ID)
		return requestRoutesResponse{Routes: itin, Err: nil}, nil
	}
}

type assignRouteRequest struct {
	ID cargo.TrackingID
	Itinerary	cargo.Itinerary
}

type assignRouteResponse struct {
	Err error	`json:"error,omitempty"`
}

func(r assignRouteResponse) error() error { return r.Err }

func makeAssignRouteEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignRouteRequest)
		err := s.AssignCargoToRoute(req.ID, req.Itinerary)
		return assignRouteResponse{Err: err}, nil
	}
}

type changeDestinationRequest struct {
	ID	cargo.TrackingID
	Destination		location.UNLcode
}

type changeDestinationResponse struct {
	Err error	`json:"error,omitempty"`
}

func(r changeDestinationResponse) error() error {return r.Err }

func makeChangeDestinationEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeDestinationRequest)
		err := s.ChangeDestination(req.ID, req.Destination)
		return changeDestinationResponse{Err: err}, nil
	}
}

type listCargosRequest struct{}

type listCargosResponse struct {
	Cargos []Cargo `json:"cargos,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r listCargosResponse) error() error { return r.Err }

func makeListCargosEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		_ = request.(listCargosRequest)
		return listCargosResponse{Cargos: s.Cargos(), Err: nil}, nil
	}
}

type listLocationsRequest struct {
}

type listLocationsResponse struct {
	Locations []Location `json:"locations,omitempty"`
	Err       error      `json:"error,omitempty"`
}

func(r listLocationsResponse) error() error { return r.Err }

func makeListLocationsEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		_ = request.(listLocationsRequest)
		return listLocationsResponse{Locations: s.Locations(), Err: nil}, nil
	}
}
// Set collects all of the endpoints that compose a booking cargo service.
type Set struct {
	BookCargoEndpoint endpoint.Endpoint
	LoadCargoEndpoint endpoint.Endpoint
	RequestRoutesEndpoint	endpoint.Endpoint
	AssignRouteEndpoint		endpoint.Endpoint
	ChangeDestinationEndpoint	endpoint.Endpoint
	ListCargosEndpoint	endpoint.Endpoint
	ListLocationsEndpoint	endpoint.Endpoint
}

// NewSet returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func NewSet(svc Service, logger log.Logger, duration metrics.Histogram, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer) Set {
	var bookCargoEndpoint endpoint.Endpoint
	{
		bookCargoEndpoint = makeBookCargoEndpoint(svc)
		
		bookCargoEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(bookCargoEndpoint)
		bookCargoEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(bookCargoEndpoint)
		bookCargoEndpoint = opentracing.TraceServer(otTracer, "BookCargo")(bookCargoEndpoint)
		if zipkinTracer != nil {
			bookCargoEndpoint = zipkin.TraceEndpoint(zipkinTracer, "BookCargo")(bookCargoEndpoint)
		}
	}

	var loadCargoEndpoint endpoint.Endpoint
	{
		loadCargoEndpoint = makeLoadCargoEndpoint(svc)
		
		loadCargoEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(loadCargoEndpoint)
		loadCargoEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(loadCargoEndpoint)
		loadCargoEndpoint = opentracing.TraceServer(otTracer, "LoadCargo")(loadCargoEndpoint)
		if zipkinTracer != nil {
			loadCargoEndpoint = zipkin.TraceEndpoint(zipkinTracer, "LoadCargo")(loadCargoEndpoint)
		}
	}

	var requestRoutesEndpoint endpoint.Endpoint
	{
		requestRoutesEndpoint = makeRequestRoutesEndpoint(svc)

		requestRoutesEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(requestRoutesEndpoint)
		requestRoutesEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(requestRoutesEndpoint)
		requestRoutesEndpoint = opentracing.TraceServer(otTracer, "RequestRoutes")(requestRoutesEndpoint)
		if zipkinTracer != nil {
			requestRoutesEndpoint = zipkin.TraceEndpoint(zipkinTracer, "RequestRoutes")(requestRoutesEndpoint)
		}
	}

	var assignRouteEndpoint endpoint.Endpoint
	{
		assignRouteEndpoint = makeAssignRouteEndpoint(svc)

		assignRouteEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(assignRouteEndpoint)
		assignRouteEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(assignRouteEndpoint)
		assignRouteEndpoint = opentracing.TraceServer(otTracer, "AssignRoute")(assignRouteEndpoint)
		if zipkinTracer != nil {
			assignRouteEndpoint = zipkin.TraceEndpoint(zipkinTracer, "AssignRoute")(assignRouteEndpoint)
		}
	}

	var changeDestinationEndpoint endpoint.Endpoint
	{
		changeDestinationEndpoint = makeChangeDestinationEndpoint(svc)

		changeDestinationEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(changeDestinationEndpoint)
		changeDestinationEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(changeDestinationEndpoint)
		changeDestinationEndpoint = opentracing.TraceServer(otTracer, "ChangeDestination")(changeDestinationEndpoint)
		if zipkinTracer != nil {
			changeDestinationEndpoint = zipkin.TraceEndpoint(zipkinTracer, "ChangeDestination")(changeDestinationEndpoint)
		}
	}

	var listCargosEndpoint endpoint.Endpoint
	{
		listCargosEndpoint = makeListCargosEndpoint(svc)

		listCargosEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(listCargosEndpoint)
		listCargosEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(listCargosEndpoint)
		listCargosEndpoint = opentracing.TraceServer(otTracer, "ListCargos")(listCargosEndpoint)
		if zipkinTracer != nil {
			listCargosEndpoint = zipkin.TraceEndpoint(zipkinTracer, "ListCargos")(listCargosEndpoint)
		}
	}

	var listLocationsEndpoint endpoint.Endpoint
	{
		listLocationsEndpoint = makeListLocationsEndpoint(svc)

		listLocationsEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Limit(1), 100))(listLocationsEndpoint)
		listLocationsEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(listLocationsEndpoint)
		listLocationsEndpoint = opentracing.TraceServer(otTracer, "ListLocations")(listLocationsEndpoint)
		if zipkinTracer != nil {
			listLocationsEndpoint = zipkin.TraceEndpoint(zipkinTracer, "ListLocations")(listLocationsEndpoint)
		}
	}

	return Set{
		BookCargoEndpoint: bookCargoEndpoint,
		LoadCargoEndpoint: loadCargoEndpoint,
		RequestRoutesEndpoint: requestRoutesEndpoint,
		AssignRouteEndpoint: assignRouteEndpoint,
		ChangeDestinationEndpoint: changeDestinationEndpoint,
		ListCargosEndpoint: listCargosEndpoint,
		ListLocationsEndpoint: listCargosEndpoint,
	}
}
// BookNewCargo implements the service interface so Set can be used as a service
func(s Set) BookNewCargo(origin location.UNLcode, destination location.UNLcode, deadline time.Time) (cargo.TrackingID, error) {
	resp, err := s.BookCargoEndpoint(context.Background(), bookCargoRequest{Origin: origin, Destination: destination, ArrivalDeadline: deadline})
	if err != nil {
		return cargo.TrackingID(""), err
	}
	response := resp.(bookCargoResponse)
	return response.ID, response.Err
}

// LoadCargo implements the service interface so Set can be used as a service
func(s Set) LoadCargo(id cargo.TrackingID) (Cargo, error) {
	resp, err := s.LoadCargoEndpoint(context.Background(), loadCargoRequest{ID: id})
	if err != nil {
		return Cargo{}, err
	}
	response := resp.(loadCargoResponse)
	return *response.Cargo, nil
}

// RequestPossibleRoutesForCargo implements the service interface so Set can be used as a service
func(s Set) RequestPossibleRoutesForCargo(id cargo.TrackingID) []cargo.Itinerary {
	resp, err := s.RequestRoutesEndpoint(context.Background(), requestRoutesRequest{ID: id})
	if err != nil {
		return []cargo.Itinerary{}
	}
	response := resp.(requestRoutesResponse)
	return response.Routes
}

// AssignCargoToRoute implements the service interface so Set can be used as a service
func(s Set) AssignCargoToRoute(id cargo.TrackingID, itinerary cargo.Itinerary) error {
	resp, err := s.AssignRouteEndpoint(context.Background(), assignRouteRequest{ID: id, Itinerary: itinerary})
	if err != nil {
		return err
	}
	response := resp.(assignRouteResponse)
	return response.Err
}

// ChangeDestination implements the service interface so Set can be used as a service
func(s Set) ChangeDestination(id cargo.TrackingID, destination location.UNLcode) error {
	resp, err := s.ChangeDestinationEndpoint(context.Background(), changeDestinationRequest{ID: id, Destination: destination})
	if err != nil {
		return err
	}
	response := resp.(changeDestinationResponse)
	return response.Err
}

// Cargos implements the service interface so Set can be used as a service
func(s Set) Cargos() []Cargo {
	resp, err := s.ListCargosEndpoint(context.Background(), listCargosRequest{})
	if err != nil {
		return []Cargo{}
	}
	response := resp.(listCargosResponse)
	return response.Cargos
}

// Locations implements the service interface so Set can be used as a service
func(s Set) Locations() []Location {
	resp, err := s.ListLocationsEndpoint(context.Background(), listLocationsRequest{})
	if err != nil {
		return []Location{}
	}
	response := resp.(listLocationsResponse)
	return response.Locations
}