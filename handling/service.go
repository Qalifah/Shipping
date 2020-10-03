package handling

import (
	"errors"
	"time"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/voyage"
	"github.com/Qalifah/shipping/location"
	"github.com/Qalifah/shipping/inspection"
)

// ErrInvalidArgument is returned when one or more arguments are invalid
var ErrInvalidArgument = errors.New("invalid argument")

// EventHandler provides a means of subscribing to registered handling events
type EventHandler interface {
	CargoWasHandled(cargo.HandlingEvent)
}

// Service provides handling operations
type Service interface {
	// RegisterHandlingEvent registers a handling event in the system, and
	// notifies interested parties that a cargo has been handled.
	RegisterHandlingEvent(completed time.Time, id cargo.TrackingID, voyageNumber voyage.Number, unLcode location.UNLcode, eventType cargo.HandlingEventType) error 
}

type service struct {
	handlingEventRespository	cargo.HandlingEventRepository
	handlingEventFactory		cargo.HandlingEventFactory
	handlingEventHandler		EventHandler
}

func(s *service) RegisterHandlingEvent(completed time.Time, id cargo.TrackingID, voyageNumber voyage.Number, unLcode location.UNLcode, eventType cargo.HandlingEventType) error {
	if completed.IsZero() || id == "" || unLcode == "" || eventType == cargo.NotHandled {
		return ErrInvalidArgument
	}

	e, err := s.handlingEventFactory.CreateHandlingEvent(time.Now(), completed, id, voyageNumber, unLcode, eventType)
	if err != nil {
		return err
	}

	s.handlingEventRespository.Store(e)
	s.handlingEventHandler.CargoWasHandled(e)

	return nil
}

// NewService creates a handling event service with necessary dependencies.
func NewService(r cargo.HandlingEventRepository, f cargo.HandlingEventFactory, h EventHandler) Service {
	return &service{
		handlingEventRespository: r,
		handlingEventFactory: f,
		handlingEventHandler: h,
	}
}

type handlingEventHandler struct {
	InspectionService	inspection.Service
}

func(h *handlingEventHandler) CargoWasHandled(event cargo.HandlingEvent) {
	h.InspectionService.InspectCargo(event.TrackingID)
}

// NewEventHandler returns a new instance of a EventHandler.
func NewEventHandler(s inspection.Service) EventHandler {
	return &handlingEventHandler{
		InspectionService: s,
	}
}