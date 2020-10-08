package tracking

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	pb "github.com/Qalifah/shipping/pb/trackingpb"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/go-kit/kit/tracing/zipkin"
	"github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

type grpcServer struct {
	trackCargo	grpctransport.Handler
}

// NewGRPCServer makes a set of endpoints available on a grpc server
func NewGRPCServer(endpoints Set, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) pb.TrackingServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
	}

	if zipkinTracer != nil {
		options = append(options, zipkin.GRPCServerTrace(zipkinTracer))
	}

	return &grpcServer{
		trackCargo: grpctransport.NewServer(
			endpoints.TrackCargoEndpoint,
			decodeGRPCTrackCargoRequest,
			encodeGRPCTrackCargoResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "trackCargo", logger)))...,
	    ),
	}
}

func(s *grpcServer) Track(ctx context.Context, req *pb.TrackRequest) (*pb.TrackReply, error) {
	_, rep, err := s.trackCargo.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.TrackReply), nil
}

// NewGRPCClient returns a booking service backed by a grpc server at the other end of the conn
func NewGRPCClient(conn *grpc.ClientConn, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) Service {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))
	var options []grpctransport.ClientOption
	if zipkinTracer != nil {
		options = append(options, zipkin.GRPCClientTrace(zipkinTracer))
	}

	var trackCargoEndpoint endpoint.Endpoint
	{
		trackCargoEndpoint = grpctransport.NewClient(
			conn,
			"pb.tracking",
			"TrackCargo",
			encodeGRPCTrackCargoRequest,
			decodeGRPCTrackCargoResponse,
			pb.TrackReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		trackCargoEndpoint = opentracing.TraceClient(otTracer, "TrackCargo")(trackCargoEndpoint)
		trackCargoEndpoint = limiter(trackCargoEndpoint)
		trackCargoEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "TrackCargo",
			Timeout: 30 * time.Second,
		}))(trackCargoEndpoint)
	}
	
	return Set{
		TrackCargoEndpoint: trackCargoEndpoint,
	}
}

func decodeGRPCTrackCargoRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.TrackRequest)
	return trackCargoRequest{ID: req.TrackingId}, nil
}

func encodeGRPCTrackCargoResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(trackCargoResponse)
	return &pb.TrackReply{Cargo: encodeCargo(*resp.Cargo), Err: err2str(resp.Err)}, nil
}

func decodeGRPCTrackCargoResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.TrackReply)
	return trackCargoResponse{Cargo: decodeCargo(reply.Cargo), Err: str2err(reply.Err)}, nil
}

func encodeGRPCTrackCargoRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(trackCargoRequest)
	return &pb.TrackRequest{TrackingId: req.ID}, nil
}

func encodeCargo(decodedCargo Cargo) *pb.Cargo {
	eta, _ := ptypes.TimestampProto(decodedCargo.ETA)
	deadline, _ := ptypes.TimestampProto(decodedCargo.ArrivalDeadline)
	encodedCargo := &pb.Cargo{
		Id: decodedCargo.TrackingID,
		StatusText: decodedCargo.StatusText,
		Origin: decodedCargo.Origin,
		Destination: decodedCargo.Destination,
		Eta: eta,
		NextExpectedActivity: decodedCargo.NextExpectedActivity,
		Deadline: deadline,
		Events: encodeEvents(decodedCargo.Events),
	}
	return encodedCargo
}

func encodeEvents(decodedEvents []Event) []*pb.Event {
	var events []*pb.Event
	for _, event := range decodedEvents {
		events = append(events, &pb.Event{Description: event.Description, Expected: event.Expected})
	}
	return events
}

func decodeCargo(encodedCargo *pb.Cargo) *Cargo {
	eta, _ := ptypes.Timestamp(encodedCargo.Eta)
	deadline, _ := ptypes.Timestamp(encodedCargo.Deadline)
	decodedCargo := &Cargo{
		TrackingID: encodedCargo.Id,
		StatusText: encodedCargo.StatusText,
		Origin: encodedCargo.Origin,
		Destination: encodedCargo.Destination,
		ETA: eta,
		NextExpectedActivity: encodedCargo.NextExpectedActivity,
		ArrivalDeadline: deadline,
        Events: decodeEvents(encodedCargo.Events),
	}
	return decodedCargo
}

func decodeEvents(encodedEvents []*pb.Event) []Event {
	var events []Event
	for _, event := range encodedEvents {
		events = append(events, Event{Description: event.Description, Expected: event.Expected})
	}
	return events
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}