package handling

import (
	"time"
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/voyage"
	"github.com/Qalifah/shipping/location"
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