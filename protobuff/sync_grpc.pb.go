// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.28.3
// source: sync.proto

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
	SyncService_SyncGetBootstrapMetadata_FullMethodName = "/qubic.archiver.archive.pb.SyncService/SyncGetBootstrapMetadata"
	SyncService_SyncGetEpochInformation_FullMethodName  = "/qubic.archiver.archive.pb.SyncService/SyncGetEpochInformation"
	SyncService_SyncGetTickInformation_FullMethodName   = "/qubic.archiver.archive.pb.SyncService/SyncGetTickInformation"
)

// SyncServiceClient is the client API for SyncService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SyncServiceClient interface {
	SyncGetBootstrapMetadata(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SyncMetadataResponse, error)
	SyncGetEpochInformation(ctx context.Context, in *SyncEpochInfoRequest, opts ...grpc.CallOption) (SyncService_SyncGetEpochInformationClient, error)
	SyncGetTickInformation(ctx context.Context, in *SyncTickInfoRequest, opts ...grpc.CallOption) (SyncService_SyncGetTickInformationClient, error)
}

type syncServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSyncServiceClient(cc grpc.ClientConnInterface) SyncServiceClient {
	return &syncServiceClient{cc}
}

func (c *syncServiceClient) SyncGetBootstrapMetadata(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SyncMetadataResponse, error) {
	out := new(SyncMetadataResponse)
	err := c.cc.Invoke(ctx, SyncService_SyncGetBootstrapMetadata_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *syncServiceClient) SyncGetEpochInformation(ctx context.Context, in *SyncEpochInfoRequest, opts ...grpc.CallOption) (SyncService_SyncGetEpochInformationClient, error) {
	stream, err := c.cc.NewStream(ctx, &SyncService_ServiceDesc.Streams[0], SyncService_SyncGetEpochInformation_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &syncServiceSyncGetEpochInformationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type SyncService_SyncGetEpochInformationClient interface {
	Recv() (*SyncEpochInfoResponse, error)
	grpc.ClientStream
}

type syncServiceSyncGetEpochInformationClient struct {
	grpc.ClientStream
}

func (x *syncServiceSyncGetEpochInformationClient) Recv() (*SyncEpochInfoResponse, error) {
	m := new(SyncEpochInfoResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *syncServiceClient) SyncGetTickInformation(ctx context.Context, in *SyncTickInfoRequest, opts ...grpc.CallOption) (SyncService_SyncGetTickInformationClient, error) {
	stream, err := c.cc.NewStream(ctx, &SyncService_ServiceDesc.Streams[1], SyncService_SyncGetTickInformation_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &syncServiceSyncGetTickInformationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type SyncService_SyncGetTickInformationClient interface {
	Recv() (*SyncTickInfoResponse, error)
	grpc.ClientStream
}

type syncServiceSyncGetTickInformationClient struct {
	grpc.ClientStream
}

func (x *syncServiceSyncGetTickInformationClient) Recv() (*SyncTickInfoResponse, error) {
	m := new(SyncTickInfoResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SyncServiceServer is the server API for SyncService service.
// All implementations must embed UnimplementedSyncServiceServer
// for forward compatibility
type SyncServiceServer interface {
	SyncGetBootstrapMetadata(context.Context, *emptypb.Empty) (*SyncMetadataResponse, error)
	SyncGetEpochInformation(*SyncEpochInfoRequest, SyncService_SyncGetEpochInformationServer) error
	SyncGetTickInformation(*SyncTickInfoRequest, SyncService_SyncGetTickInformationServer) error
	mustEmbedUnimplementedSyncServiceServer()
}

// UnimplementedSyncServiceServer must be embedded to have forward compatible implementations.
type UnimplementedSyncServiceServer struct {
}

func (UnimplementedSyncServiceServer) SyncGetBootstrapMetadata(context.Context, *emptypb.Empty) (*SyncMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SyncGetBootstrapMetadata not implemented")
}
func (UnimplementedSyncServiceServer) SyncGetEpochInformation(*SyncEpochInfoRequest, SyncService_SyncGetEpochInformationServer) error {
	return status.Errorf(codes.Unimplemented, "method SyncGetEpochInformation not implemented")
}
func (UnimplementedSyncServiceServer) SyncGetTickInformation(*SyncTickInfoRequest, SyncService_SyncGetTickInformationServer) error {
	return status.Errorf(codes.Unimplemented, "method SyncGetTickInformation not implemented")
}
func (UnimplementedSyncServiceServer) mustEmbedUnimplementedSyncServiceServer() {}

// UnsafeSyncServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SyncServiceServer will
// result in compilation errors.
type UnsafeSyncServiceServer interface {
	mustEmbedUnimplementedSyncServiceServer()
}

func RegisterSyncServiceServer(s grpc.ServiceRegistrar, srv SyncServiceServer) {
	s.RegisterService(&SyncService_ServiceDesc, srv)
}

func _SyncService_SyncGetBootstrapMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SyncServiceServer).SyncGetBootstrapMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SyncService_SyncGetBootstrapMetadata_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SyncServiceServer).SyncGetBootstrapMetadata(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SyncService_SyncGetEpochInformation_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SyncEpochInfoRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SyncServiceServer).SyncGetEpochInformation(m, &syncServiceSyncGetEpochInformationServer{stream})
}

type SyncService_SyncGetEpochInformationServer interface {
	Send(*SyncEpochInfoResponse) error
	grpc.ServerStream
}

type syncServiceSyncGetEpochInformationServer struct {
	grpc.ServerStream
}

func (x *syncServiceSyncGetEpochInformationServer) Send(m *SyncEpochInfoResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _SyncService_SyncGetTickInformation_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SyncTickInfoRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SyncServiceServer).SyncGetTickInformation(m, &syncServiceSyncGetTickInformationServer{stream})
}

type SyncService_SyncGetTickInformationServer interface {
	Send(*SyncTickInfoResponse) error
	grpc.ServerStream
}

type syncServiceSyncGetTickInformationServer struct {
	grpc.ServerStream
}

func (x *syncServiceSyncGetTickInformationServer) Send(m *SyncTickInfoResponse) error {
	return x.ServerStream.SendMsg(m)
}

// SyncService_ServiceDesc is the grpc.ServiceDesc for SyncService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SyncService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "qubic.archiver.archive.pb.SyncService",
	HandlerType: (*SyncServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SyncGetBootstrapMetadata",
			Handler:    _SyncService_SyncGetBootstrapMetadata_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SyncGetEpochInformation",
			Handler:       _SyncService_SyncGetEpochInformation_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SyncGetTickInformation",
			Handler:       _SyncService_SyncGetTickInformation_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "sync.proto",
}

const (
	SyncClientService_SyncGetStatus_FullMethodName = "/qubic.archiver.archive.pb.SyncClientService/SyncGetStatus"
)

// SyncClientServiceClient is the client API for SyncClientService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SyncClientServiceClient interface {
	SyncGetStatus(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SyncStatus, error)
}

type syncClientServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSyncClientServiceClient(cc grpc.ClientConnInterface) SyncClientServiceClient {
	return &syncClientServiceClient{cc}
}

func (c *syncClientServiceClient) SyncGetStatus(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SyncStatus, error) {
	out := new(SyncStatus)
	err := c.cc.Invoke(ctx, SyncClientService_SyncGetStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SyncClientServiceServer is the server API for SyncClientService service.
// All implementations must embed UnimplementedSyncClientServiceServer
// for forward compatibility
type SyncClientServiceServer interface {
	SyncGetStatus(context.Context, *emptypb.Empty) (*SyncStatus, error)
	mustEmbedUnimplementedSyncClientServiceServer()
}

// UnimplementedSyncClientServiceServer must be embedded to have forward compatible implementations.
type UnimplementedSyncClientServiceServer struct {
}

func (UnimplementedSyncClientServiceServer) SyncGetStatus(context.Context, *emptypb.Empty) (*SyncStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SyncGetStatus not implemented")
}
func (UnimplementedSyncClientServiceServer) mustEmbedUnimplementedSyncClientServiceServer() {}

// UnsafeSyncClientServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SyncClientServiceServer will
// result in compilation errors.
type UnsafeSyncClientServiceServer interface {
	mustEmbedUnimplementedSyncClientServiceServer()
}

func RegisterSyncClientServiceServer(s grpc.ServiceRegistrar, srv SyncClientServiceServer) {
	s.RegisterService(&SyncClientService_ServiceDesc, srv)
}

func _SyncClientService_SyncGetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SyncClientServiceServer).SyncGetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SyncClientService_SyncGetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SyncClientServiceServer).SyncGetStatus(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// SyncClientService_ServiceDesc is the grpc.ServiceDesc for SyncClientService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SyncClientService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "qubic.archiver.archive.pb.SyncClientService",
	HandlerType: (*SyncClientServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SyncGetStatus",
			Handler:    _SyncClientService_SyncGetStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "sync.proto",
}
