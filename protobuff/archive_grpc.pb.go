// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.23.4
// source: archive.proto

package protobuff

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ArchiveService_GetTickData_FullMethodName                    = "/qubic.archiver.archive.pb.ArchiveService/GetTickData"
	ArchiveService_GetQuorumTickData_FullMethodName              = "/qubic.archiver.archive.pb.ArchiveService/GetQuorumTickData"
	ArchiveService_GetTickTransactions_FullMethodName            = "/qubic.archiver.archive.pb.ArchiveService/GetTickTransactions"
	ArchiveService_GetTickTransferTransactions_FullMethodName    = "/qubic.archiver.archive.pb.ArchiveService/GetTickTransferTransactions"
	ArchiveService_GetChainHash_FullMethodName                   = "/qubic.archiver.archive.pb.ArchiveService/GetChainHash"
	ArchiveService_GetTransaction_FullMethodName                 = "/qubic.archiver.archive.pb.ArchiveService/GetTransaction"
	ArchiveService_GetTransferTransactionsPerTick_FullMethodName = "/qubic.archiver.archive.pb.ArchiveService/GetTransferTransactionsPerTick"
	ArchiveService_GetComputors_FullMethodName                   = "/qubic.archiver.archive.pb.ArchiveService/GetComputors"
	ArchiveService_GetStatus_FullMethodName                      = "/qubic.archiver.archive.pb.ArchiveService/GetStatus"
	ArchiveService_GetBlockHeight_FullMethodName                 = "/qubic.archiver.archive.pb.ArchiveService/GetBlockHeight"
)

// ArchiveServiceClient is the client API for ArchiveService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ArchiveServiceClient interface {
	GetTickData(ctx context.Context, in *GetTickDataRequest, opts ...grpc.CallOption) (*GetTickDataResponse, error)
	GetQuorumTickData(ctx context.Context, in *GetQuorumTickDataRequest, opts ...grpc.CallOption) (*GetQuorumTickDataResponse, error)
	GetTickTransactions(ctx context.Context, in *GetTickTransactionsRequest, opts ...grpc.CallOption) (*GetTickTransactionsResponse, error)
	GetTickTransferTransactions(ctx context.Context, in *GetTickTransactionsRequest, opts ...grpc.CallOption) (*GetTickTransactionsResponse, error)
	GetChainHash(ctx context.Context, in *GetChainHashRequest, opts ...grpc.CallOption) (*GetChainHashResponse, error)
	GetTransaction(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error)
	GetTransferTransactionsPerTick(ctx context.Context, in *GetTransferTransactionsPerTickRequest, opts ...grpc.CallOption) (*GetTransferTransactionsPerTickResponse, error)
	GetComputors(ctx context.Context, in *GetComputorsRequest, opts ...grpc.CallOption) (*GetComputorsResponse, error)
	GetStatus(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetStatusResponse, error)
	GetBlockHeight(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetBlockHeightResponse, error)
}

type archiveServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewArchiveServiceClient(cc grpc.ClientConnInterface) ArchiveServiceClient {
	return &archiveServiceClient{cc}
}

func (c *archiveServiceClient) GetTickData(ctx context.Context, in *GetTickDataRequest, opts ...grpc.CallOption) (*GetTickDataResponse, error) {
	out := new(GetTickDataResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetTickData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetQuorumTickData(ctx context.Context, in *GetQuorumTickDataRequest, opts ...grpc.CallOption) (*GetQuorumTickDataResponse, error) {
	out := new(GetQuorumTickDataResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetQuorumTickData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetTickTransactions(ctx context.Context, in *GetTickTransactionsRequest, opts ...grpc.CallOption) (*GetTickTransactionsResponse, error) {
	out := new(GetTickTransactionsResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetTickTransactions_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetTickTransferTransactions(ctx context.Context, in *GetTickTransactionsRequest, opts ...grpc.CallOption) (*GetTickTransactionsResponse, error) {
	out := new(GetTickTransactionsResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetTickTransferTransactions_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetChainHash(ctx context.Context, in *GetChainHashRequest, opts ...grpc.CallOption) (*GetChainHashResponse, error) {
	out := new(GetChainHashResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetChainHash_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetTransaction(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error) {
	out := new(GetTransactionResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetTransaction_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetTransferTransactionsPerTick(ctx context.Context, in *GetTransferTransactionsPerTickRequest, opts ...grpc.CallOption) (*GetTransferTransactionsPerTickResponse, error) {
	out := new(GetTransferTransactionsPerTickResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetTransferTransactionsPerTick_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetComputors(ctx context.Context, in *GetComputorsRequest, opts ...grpc.CallOption) (*GetComputorsResponse, error) {
	out := new(GetComputorsResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetComputors_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetStatus(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetStatusResponse, error) {
	out := new(GetStatusResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *archiveServiceClient) GetBlockHeight(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetBlockHeightResponse, error) {
	out := new(GetBlockHeightResponse)
	err := c.cc.Invoke(ctx, ArchiveService_GetBlockHeight_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ArchiveServiceServer is the server API for ArchiveService service.
// All implementations must embed UnimplementedArchiveServiceServer
// for forward compatibility
type ArchiveServiceServer interface {
	GetTickData(context.Context, *GetTickDataRequest) (*GetTickDataResponse, error)
	GetQuorumTickData(context.Context, *GetQuorumTickDataRequest) (*GetQuorumTickDataResponse, error)
	GetTickTransactions(context.Context, *GetTickTransactionsRequest) (*GetTickTransactionsResponse, error)
	GetTickTransferTransactions(context.Context, *GetTickTransactionsRequest) (*GetTickTransactionsResponse, error)
	GetChainHash(context.Context, *GetChainHashRequest) (*GetChainHashResponse, error)
	GetTransaction(context.Context, *GetTransactionRequest) (*GetTransactionResponse, error)
	GetTransferTransactionsPerTick(context.Context, *GetTransferTransactionsPerTickRequest) (*GetTransferTransactionsPerTickResponse, error)
	GetComputors(context.Context, *GetComputorsRequest) (*GetComputorsResponse, error)
	GetStatus(context.Context, *emptypb.Empty) (*GetStatusResponse, error)
	GetBlockHeight(context.Context, *emptypb.Empty) (*GetBlockHeightResponse, error)
	mustEmbedUnimplementedArchiveServiceServer()
}

// UnimplementedArchiveServiceServer must be embedded to have forward compatible implementations.
type UnimplementedArchiveServiceServer struct {
}

func (UnimplementedArchiveServiceServer) GetTickData(context.Context, *GetTickDataRequest) (*GetTickDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTickData not implemented")
}
func (UnimplementedArchiveServiceServer) GetQuorumTickData(context.Context, *GetQuorumTickDataRequest) (*GetQuorumTickDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetQuorumTickData not implemented")
}
func (UnimplementedArchiveServiceServer) GetTickTransactions(context.Context, *GetTickTransactionsRequest) (*GetTickTransactionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTickTransactions not implemented")
}
func (UnimplementedArchiveServiceServer) GetTickTransferTransactions(context.Context, *GetTickTransactionsRequest) (*GetTickTransactionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTickTransferTransactions not implemented")
}
func (UnimplementedArchiveServiceServer) GetChainHash(context.Context, *GetChainHashRequest) (*GetChainHashResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetChainHash not implemented")
}
func (UnimplementedArchiveServiceServer) GetTransaction(context.Context, *GetTransactionRequest) (*GetTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTransaction not implemented")
}
func (UnimplementedArchiveServiceServer) GetTransferTransactionsPerTick(context.Context, *GetTransferTransactionsPerTickRequest) (*GetTransferTransactionsPerTickResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTransferTransactionsPerTick not implemented")
}
func (UnimplementedArchiveServiceServer) GetComputors(context.Context, *GetComputorsRequest) (*GetComputorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetComputors not implemented")
}
func (UnimplementedArchiveServiceServer) GetStatus(context.Context, *emptypb.Empty) (*GetStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedArchiveServiceServer) GetBlockHeight(context.Context, *emptypb.Empty) (*GetBlockHeightResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBlockHeight not implemented")
}
func (UnimplementedArchiveServiceServer) mustEmbedUnimplementedArchiveServiceServer() {}

// UnsafeArchiveServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ArchiveServiceServer will
// result in compilation errors.
type UnsafeArchiveServiceServer interface {
	mustEmbedUnimplementedArchiveServiceServer()
}

func RegisterArchiveServiceServer(s grpc.ServiceRegistrar, srv ArchiveServiceServer) {
	s.RegisterService(&ArchiveService_ServiceDesc, srv)
}

func _ArchiveService_GetTickData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTickDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetTickData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetTickData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetTickData(ctx, req.(*GetTickDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetQuorumTickData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetQuorumTickDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetQuorumTickData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetQuorumTickData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetQuorumTickData(ctx, req.(*GetQuorumTickDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetTickTransactions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTickTransactionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetTickTransactions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetTickTransactions_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetTickTransactions(ctx, req.(*GetTickTransactionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetTickTransferTransactions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTickTransactionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetTickTransferTransactions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetTickTransferTransactions_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetTickTransferTransactions(ctx, req.(*GetTickTransactionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetChainHash_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetChainHashRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetChainHash(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetChainHash_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetChainHash(ctx, req.(*GetChainHashRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetTransaction_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetTransaction(ctx, req.(*GetTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetTransferTransactionsPerTick_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTransferTransactionsPerTickRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetTransferTransactionsPerTick(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetTransferTransactionsPerTick_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetTransferTransactionsPerTick(ctx, req.(*GetTransferTransactionsPerTickRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetComputors_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetComputorsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetComputors(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetComputors_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetComputors(ctx, req.(*GetComputorsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetStatus(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArchiveService_GetBlockHeight_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArchiveServiceServer).GetBlockHeight(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArchiveService_GetBlockHeight_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArchiveServiceServer).GetBlockHeight(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// ArchiveService_ServiceDesc is the grpc.ServiceDesc for ArchiveService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ArchiveService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "qubic.archiver.archive.pb.ArchiveService",
	HandlerType: (*ArchiveServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetTickData",
			Handler:    _ArchiveService_GetTickData_Handler,
		},
		{
			MethodName: "GetQuorumTickData",
			Handler:    _ArchiveService_GetQuorumTickData_Handler,
		},
		{
			MethodName: "GetTickTransactions",
			Handler:    _ArchiveService_GetTickTransactions_Handler,
		},
		{
			MethodName: "GetTickTransferTransactions",
			Handler:    _ArchiveService_GetTickTransferTransactions_Handler,
		},
		{
			MethodName: "GetChainHash",
			Handler:    _ArchiveService_GetChainHash_Handler,
		},
		{
			MethodName: "GetTransaction",
			Handler:    _ArchiveService_GetTransaction_Handler,
		},
		{
			MethodName: "GetTransferTransactionsPerTick",
			Handler:    _ArchiveService_GetTransferTransactionsPerTick_Handler,
		},
		{
			MethodName: "GetComputors",
			Handler:    _ArchiveService_GetComputors_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _ArchiveService_GetStatus_Handler,
		},
		{
			MethodName: "GetBlockHeight",
			Handler:    _ArchiveService_GetBlockHeight_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "archive.proto",
}
