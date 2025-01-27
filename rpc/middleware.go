package rpc

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/validator/tick"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) tickNumberInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	var err error

	switch request := req.(type) {

	case *protobuff.GetTickRequestV2:
		err = s.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickTransactionsRequest:
		err = s.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickDataRequest:
		err = s.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetQuorumTickDataRequest:
		err = s.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickApprovedTransactionsRequest:
		err = s.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetChainHashRequest:
		err = s.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickTransactionsRequestV2:
		err = s.checkTickWithinArchiverIntervals(ctx, request.TickNumber)

	default:
		break
	}

	if err != nil {
		return nil, errors.Wrapf(err, "invalid tick number")
	}

	h, err := handler(ctx, req)

	return h, err
}

func (s *Server) checkTickWithinArchiverIntervals(ctx context.Context, tickNumber uint32) error {

	lastProcessedTick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get last processed tick")
	}

	if tickNumber > lastProcessedTick.TickNumber {
		st := status.Newf(codes.FailedPrecondition, "requested tick number %d is greater than last processed tick %d", tickNumber, lastProcessedTick.TickNumber)
		st, err = st.WithDetails(&protobuff.LastProcessedTick{LastProcessedTick: lastProcessedTick.TickNumber})
		if err != nil {
			return status.Errorf(codes.Internal, "creating custom status")
		}
		return st.Err()
	}

	processedTickIntervalsPerEpoch, err := s.store.GetProcessedTickIntervals(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "getting processed tick intervals per epoch")
	}

	wasSkipped, nextAvailableTick := tick.WasSkippedByArchive(tickNumber, processedTickIntervalsPerEpoch)
	if wasSkipped == true {
		st := status.Newf(codes.OutOfRange, "provided tick number %d was skipped by the system, next available tick is %d", tickNumber, nextAvailableTick)
		st, err = st.WithDetails(&protobuff.NextAvailableTick{NextTickNumber: nextAvailableTick})
		if err != nil {
			return status.Errorf(codes.Internal, "creating custom status")
		}
		return st.Err()
	}

	return nil
}
