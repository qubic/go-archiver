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

type InterceptorStore interface {
	GetLastProcessedTick(ctx context.Context) (*protobuff.ProcessedTick, error)
	GetProcessedTickIntervals(ctx context.Context) ([]*protobuff.ProcessedTickIntervalsPerEpoch, error)
}

type TickWithinBoundsInterceptor struct {
	store InterceptorStore
}

func NewTickWithinBoundsInterceptor(store InterceptorStore) *TickWithinBoundsInterceptor {
	return &TickWithinBoundsInterceptor{store: store}
}

func (twb *TickWithinBoundsInterceptor) GetInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	var err error

	switch request := req.(type) {

	case *protobuff.GetTickRequestV2:
		err = twb.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickTransactionsRequest:
		err = twb.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickDataRequest:
		err = twb.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetQuorumTickDataRequest:
		err = twb.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickApprovedTransactionsRequest:
		err = twb.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetChainHashRequest:
		err = twb.checkTickWithinArchiverIntervals(ctx, request.TickNumber)
	case *protobuff.GetTickTransactionsRequestV2:
		err = twb.checkTickWithinArchiverIntervals(ctx, request.TickNumber)

	default:
		break
	}

	if err != nil {
		return nil, errors.Wrapf(err, "invalid tick number")
	}

	h, err := handler(ctx, req)

	return h, err
}

func (twb *TickWithinBoundsInterceptor) checkTickWithinArchiverIntervals(ctx context.Context, tickNumber uint32) error {

	lastProcessedTick, err := twb.store.GetLastProcessedTick(ctx)
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

	processedTickIntervalsPerEpoch, err := twb.store.GetProcessedTickIntervals(ctx)
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
