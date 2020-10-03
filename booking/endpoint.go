package booking

import (
	"context"
	"time"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/location"

	"github.com/go-kit/kit/endpoint"
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

func makeListLocationsEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		_ = request.(listLocationsRequest)
		return listLocationsResponse{Locations: s.Locations(), Err: nil}, nil
	}
}