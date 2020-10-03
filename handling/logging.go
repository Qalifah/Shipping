package handling

import (
	"time"

	"github.com/go-kit/kit/log"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/location"
	"github.com/Qalifah/shipping/voyage"
)

type loggingService struct {
	logger log.Logger
	Service
}

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(logger log.Logger, s Service) Service {
	return &loggingService{logger, s}
}

func (s *loggingService) RegisterHandlingEvent(completed time.Time, id cargo.TrackingID, voyageNumber voyage.Number, unLcode location.UNLcode, eventType cargo.HandlingEventType) (err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "register_incident",
			"tracking_id", id,
			"location", unLcode,
			"voyage", voyageNumber,
			"event_type", eventType,
			"completion_time", completed,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.Service.RegisterHandlingEvent(completed, id, voyageNumber, unLcode, eventType)
}