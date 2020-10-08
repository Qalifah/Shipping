package handling

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/location"
	pb "github.com/Qalifah/shipping/pb/handlingpb"
	"github.com/Qalifah/shipping/voyage"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/go-kit/kit/tracing/zipkin"
	"github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"

	"github.com/golang/protobuf/ptypes"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

type grpcServer struct {
	registerEvent	grpctransport.Handler
}

// NewGRPCServer makes a set of endpoints available on a grpc server
func NewGRPCServer(endpoints Set, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) pb.HandlingServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
	}

	if zipkinTracer != nil {
		options = append(options, zipkin.GRPCServerTrace(zipkinTracer))
	}

	return &grpcServer{
		registerEvent: grpctransport.NewServer(
			endpoints.RegisterEventEndpoint,
			decodeGRPCRegisterEventRequest,
			encodeGRPCRegisterEventResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "bookCargo", logger)))...,
		),
	}
}

func(s *grpcServer) RegisterHandlingEvent(ctx context.Context, req *pb.RegisterHandlingEventRequest) (*pb.RegisterHandlingEventReply, error) {
	_, rep, err := s.registerEvent.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.RegisterHandlingEventReply), nil 
}

// NewGRPCClient returns a booking service backed by a grpc server at the other end of the conn
func NewGRPCClient(conn *grpc.ClientConn, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) Service {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))
	var options []grpctransport.ClientOption
	if zipkinTracer != nil {
		options = append(options, zipkin.GRPCClientTrace(zipkinTracer))
	}
	var registerEventEndpoint	endpoint.Endpoint
	{
		registerEventEndpoint = grpctransport.NewClient(
			conn,
			"pb.Handling",
			"RegisterEvent",
			encodeGRPCRegisterEventRequest,
			decodeGRPCRegisterEventResponse,
			pb.RegisterHandlingEventReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		registerEventEndpoint = opentracing.TraceClient(otTracer, "RegisterHandlingEvent")(registerEventEndpoint)
		registerEventEndpoint = limiter(registerEventEndpoint)
		registerEventEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "RegisterHandlingEvent",
			Timeout: 30 * time.Second,
		}))(registerEventEndpoint)
	}
	return Set{
		RegisterEventEndpoint: registerEventEndpoint,
	}
}

func decodeGRPCRegisterEventRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.RegisterHandlingEventRequest)
	completionTime, _ := ptypes.Timestamp(req.Completed)
	return registerEventRequest{
		ID: cargo.TrackingID(req.TrackingId), 
		Location: location.UNLcode(req.Location), 
		Voyage: voyage.Number(req.VoyageNumber), 
		EventType: cargo.HandlingEventType(req.EventType), 
		CompletionTime: completionTime}, nil
}

func encodeGRPCRegisterEventResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(registerEventResponse)
	return &pb.RegisterHandlingEventReply{Err: err2str(resp.Err)}, nil
}

func encodeGRPCRegisterEventRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(registerEventRequest)
	completed, _ := ptypes.TimestampProto(req.CompletionTime)
	return &pb.RegisterHandlingEventRequest{
		Completed: completed,
		TrackingId: string(req.ID),
		VoyageNumber: string(req.Voyage),
		Location: string(req.Location),
		EventType: int64(req.EventType),
	}, nil
}

func decodeGRPCRegisterEventResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.RegisterHandlingEventReply)
	return registerEventResponse{Err: str2err(reply.Err)}, nil
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