package cargo

import (
	"errors"
	"time"

	"github.com/Qalifah/shipping/location"
	"github.com/Qalifah/shipping/voyage"
)

// HandlingEventType describes the type of a handling event
type HandlingEventType int

// HandlingActivity represents how and where a cargo can be handled, and can
// be used to express predictions about what is expected to happen to a cargo
// in the future.
type HandlingActivity struct {
	Type	HandlingEventType
	Location	location.UNLcode
	VoyageNumber	voyage.Number
}

// HandlingEvent is used to register the event when, for instance, a cargo is
// unloaded from a carrier at a some location at a given time.
type HandlingEvent struct {
	TrackingID TrackingID
	Activity HandlingActivity
}

// valid handling event types
const (
	NotHandled HandlingEventType = iota
	Load
	Unload
	Receive
	Claim
	Customs
)

func (t HandlingEventType) String() string {
	switch t {
	case NotHandled:
		return "Not Handled"
	case Load:
		return "Load"
	case Unload:
		return "Unload"
	case Receive:
		return "Receive"
	case Claim:
		return "Claim"
	case Customs:
		return "Customs"
	}
	return ""
}

// HandlingHistory is the handling history of a cargo
type HandlingHistory struct {
	HandlingEvents []HandlingEvent
}

// MostRecentlyCompletedEvent returns the most recently completed handling event
func(h HandlingHistory) MostRecentlyCompletedEvent() (HandlingEvent, error) {
	if len(h.HandlingEvents) == 0 {
		return HandlingEvent{}, errors.New("delivery history is empty")
	}
	return h.HandlingEvents[len(h.HandlingEvents)-1], nil
}

// HandlingEventRepository provides access to the handling event store
type HandlingEventRepository interface {
	Store(e HandlingEvent)
	QueryHandlingHistory(TrackingID) HandlingHistory
}

// HandlingEventFactory creates handling events
type HandlingEventFactory struct {
	CargoRepository Repository
	VoyageRepository voyage.Repository
	LocationRepository	location.Repository
}

// CreateHandlingEvent creates a validated handling event
func(f *HandlingEventFactory) CreateHandlingEvent(registered time.Time, completed time.Time, id TrackingID, voyageNumber voyage.Number, unlCode location.UNLcode, eventType HandlingEventType) (HandlingEvent, error) {
	if _, err := f.CargoRepository.Find(id); err != nil {
		return HandlingEvent{}, err
	}

	if _, err := f.VoyageRepository.Find(voyageNumber); err != nil {
		if len(voyageNumber) > 0 {
			return HandlingEvent{}, err
		}
	}

	if _, err := f.LocationRepository.Find(unlCode); err != nil {
		return HandlingEvent{}, err
	}

	return HandlingEvent{
		TrackingID: id,
		Activity: HandlingActivity{
			Type: eventType,
			Location: unlCode,
			VoyageNumber: voyageNumber,
		},
	}, nil
}