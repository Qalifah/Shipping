package cargo

import (
	"time"
	"github.com/Qalifah/shipping/location"
	"github.com/Qalifah/shipping/voyage"
)

// Leg describes the transportation between two locations on a voyage
type Leg struct {
	VoyageNumber	voyage.Number	`json:"voyage_number"`
	LoadLocation	location.UNLcode	`json:"from"`
	UnLoadLocation	location.UNLcode	`json:"to"`
	LoadTime		time.Time			`json:"load_time"`
	UnLoadTime		time.Time			`json:"unload_time"`
}

// NewLeg creates a new itinerary leg
func NewLeg(voyageNumber voyage.Number, loadLocation, unloadLocation location.UNLcode, loadTime, unloadTime time.Time) Leg {
	return Leg{
		VoyageNumber: voyageNumber,
		LoadLocation: loadLocation,
		UnLoadLocation: unloadLocation,
		LoadTime: loadTime,
		UnLoadTime: unloadTime,
	}
}

// Itinerary specifies steps required to transport a cargo from its origin to
// destination.
type Itinerary struct {
	Legs []Leg `json:"legs"`
}

// IsEmpty checks if the itinerary contains at least one leg
func (i Itinerary) IsEmpty() bool {
	return i.Legs == nil || len(i.Legs) == 0
}

// InitialDepartureLocation returns the start of the itinerary
func (i Itinerary) InitialDepartureLocation() location.UNLcode {
	if i.IsEmpty() {
		return location.UNLcode("")
	}
	return i.Legs[0].LoadLocation
}

// FinalArrivalLocation returns the end of the itinerary
func (i Itinerary) FinalArrivalLocation() location.UNLcode {
	return i.Legs[len(i.Legs)-1].UnLoadLocation
}

// FinalArrivalTime returns the expected arrival time at final destination
func (i Itinerary) FinalArrivalTime() time.Time {
	return i.Legs[len(i.Legs)-1].UnLoadTime
}

// IsExpected checks if the given handling event is expected when executing
// this itinerary.
func (i Itinerary) IsExpected(event HandlingEvent) bool {
	if i.IsEmpty() {
		return true
	}
	switch  event.Activity.Type {
	case Receive: 
		return i.InitialDepartureLocation() == event.Activity.Location
	case Load:
		for _, l := range i.Legs {
			if l.LoadLocation == event.Activity.Location && l.VoyageNumber == event.Activity.VoyageNumber {
				return true
			}
		}
		return false
	case Unload:
		for _, l := range i.Legs {
			if l.UnLoadLocation == event.Activity.Location && l.VoyageNumber == event.Activity.VoyageNumber {
				return true
			}
		}
		return false
	case Claim:
		return i.FinalArrivalLocation() == event.Activity.Location
	}
	return true
}