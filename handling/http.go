package handling

import (
	"net/http"
	"context"
	"time"
	"encoding/json"

	"github.com/gorilla/mux"

	kitlog "github.com/go-kit/kit/log"
	kithttp	"github.com/go-kit/kit/transport/http"
	"github.com/go-kit/kit/transport"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/location"
	"github.com/Qalifah/shipping/voyage"
)

// MakeHandler returns a new handler for the handling service 
func MakeHandler(s Service, logger kitlog.Logger) http.Handler {
	r := mux.NewRouter()

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder(encodeError),
	}

	registerEventHandler := kithttp.NewServer(
		makeRegisterEventEndpoint(s),
		decodeRegisterEventRequest,
		encodeResponse,
		opts...,
	)

	r.Handle("/handling/v1/events", registerEventHandler).Methods("POST")

	return r
}

func decodeRegisterEventRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var body struct {
		CompletionTime time.Time `json:"completion_time"`
		TrackingID     string    `json:"tracking_id"`
		VoyageNumber   string    `json:"voyage"`
		Location       string    `json:"location"`
		EventType      string    `json:"event_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}

	return registerEventRequest{
		ID:		cargo.TrackingID(body.TrackingID),
		Location:	location.UNLcode(body.Location),
		Voyage: 	voyage.Number(body.VoyageNumber),
		EventType:	stringToEventType(body.EventType),
		CompletionTime : body.CompletionTime,
	}, nil
}

func stringToEventType(s string) cargo.HandlingEventType {
	types := map[string]cargo.HandlingEventType{
		cargo.Receive.String(): cargo.Receive,
		cargo.Load.String():    cargo.Load,
		cargo.Unload.String():  cargo.Unload,
		cargo.Customs.String(): cargo.Customs,
		cargo.Claim.String():   cargo.Claim,
	}
	return types[s]
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

type errorer interface {
	error() error

} 

// encode errors from business-logic
func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	case cargo.ErrUnknown:
		w.WriteHeader(http.StatusNotFound)
	case ErrInvalidArgument:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}