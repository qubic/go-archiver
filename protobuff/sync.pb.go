// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.1
// 	protoc        v5.28.3
// source: sync.proto

package protobuff

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type SyncEpochData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ComputorList                   *Computors                           `protobuf:"bytes,1,opt,name=computor_list,json=computorList,proto3" json:"computor_list,omitempty"`
	LastTickQuorumDataPerIntervals *LastTickQuorumDataPerEpochIntervals `protobuf:"bytes,2,opt,name=last_tick_quorum_data_per_intervals,json=lastTickQuorumDataPerIntervals,proto3" json:"last_tick_quorum_data_per_intervals,omitempty"`
}

func (x *SyncEpochData) Reset() {
	*x = SyncEpochData{}
	mi := &file_sync_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncEpochData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncEpochData) ProtoMessage() {}

func (x *SyncEpochData) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncEpochData.ProtoReflect.Descriptor instead.
func (*SyncEpochData) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{0}
}

func (x *SyncEpochData) GetComputorList() *Computors {
	if x != nil {
		return x.ComputorList
	}
	return nil
}

func (x *SyncEpochData) GetLastTickQuorumDataPerIntervals() *LastTickQuorumDataPerEpochIntervals {
	if x != nil {
		return x.LastTickQuorumDataPerIntervals
	}
	return nil
}

type SyncTickData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TickData           *TickData            `protobuf:"bytes,1,opt,name=tick_data,json=tickData,proto3" json:"tick_data,omitempty"`
	QuorumData         *QuorumTickData      `protobuf:"bytes,2,opt,name=quorum_data,json=quorumData,proto3" json:"quorum_data,omitempty"`
	Transactions       []*Transaction       `protobuf:"bytes,3,rep,name=transactions,proto3" json:"transactions,omitempty"`
	TransactionsStatus []*TransactionStatus `protobuf:"bytes,4,rep,name=transactions_status,json=transactionsStatus,proto3" json:"transactions_status,omitempty"`
}

func (x *SyncTickData) Reset() {
	*x = SyncTickData{}
	mi := &file_sync_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncTickData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncTickData) ProtoMessage() {}

func (x *SyncTickData) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncTickData.ProtoReflect.Descriptor instead.
func (*SyncTickData) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{1}
}

func (x *SyncTickData) GetTickData() *TickData {
	if x != nil {
		return x.TickData
	}
	return nil
}

func (x *SyncTickData) GetQuorumData() *QuorumTickData {
	if x != nil {
		return x.QuorumData
	}
	return nil
}

func (x *SyncTickData) GetTransactions() []*Transaction {
	if x != nil {
		return x.Transactions
	}
	return nil
}

func (x *SyncTickData) GetTransactionsStatus() []*TransactionStatus {
	if x != nil {
		return x.TransactionsStatus
	}
	return nil
}

type SyncMetadataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ArchiverVersion        string                            `protobuf:"bytes,1,opt,name=archiver_version,json=archiverVersion,proto3" json:"archiver_version,omitempty"`
	MaxObjectRequest       int32                             `protobuf:"varint,2,opt,name=maxObjectRequest,proto3" json:"maxObjectRequest,omitempty"`
	ProcessedTickIntervals []*ProcessedTickIntervalsPerEpoch `protobuf:"bytes,3,rep,name=processed_tick_intervals,json=processedTickIntervals,proto3" json:"processed_tick_intervals,omitempty"` //repeated SkippedTicksInterval skipped_tick_intervals = 4;
}

func (x *SyncMetadataResponse) Reset() {
	*x = SyncMetadataResponse{}
	mi := &file_sync_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncMetadataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncMetadataResponse) ProtoMessage() {}

func (x *SyncMetadataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncMetadataResponse.ProtoReflect.Descriptor instead.
func (*SyncMetadataResponse) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{2}
}

func (x *SyncMetadataResponse) GetArchiverVersion() string {
	if x != nil {
		return x.ArchiverVersion
	}
	return ""
}

func (x *SyncMetadataResponse) GetMaxObjectRequest() int32 {
	if x != nil {
		return x.MaxObjectRequest
	}
	return 0
}

func (x *SyncMetadataResponse) GetProcessedTickIntervals() []*ProcessedTickIntervalsPerEpoch {
	if x != nil {
		return x.ProcessedTickIntervals
	}
	return nil
}

type SyncEpochInfoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Epochs []uint32 `protobuf:"varint,1,rep,packed,name=epochs,proto3" json:"epochs,omitempty"`
}

func (x *SyncEpochInfoRequest) Reset() {
	*x = SyncEpochInfoRequest{}
	mi := &file_sync_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncEpochInfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncEpochInfoRequest) ProtoMessage() {}

func (x *SyncEpochInfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncEpochInfoRequest.ProtoReflect.Descriptor instead.
func (*SyncEpochInfoRequest) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{3}
}

func (x *SyncEpochInfoRequest) GetEpochs() []uint32 {
	if x != nil {
		return x.Epochs
	}
	return nil
}

type SyncEpochInfoResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Epochs []*SyncEpochData `protobuf:"bytes,1,rep,name=epochs,proto3" json:"epochs,omitempty"`
}

func (x *SyncEpochInfoResponse) Reset() {
	*x = SyncEpochInfoResponse{}
	mi := &file_sync_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncEpochInfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncEpochInfoResponse) ProtoMessage() {}

func (x *SyncEpochInfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncEpochInfoResponse.ProtoReflect.Descriptor instead.
func (*SyncEpochInfoResponse) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{4}
}

func (x *SyncEpochInfoResponse) GetEpochs() []*SyncEpochData {
	if x != nil {
		return x.Epochs
	}
	return nil
}

type SyncTickInfoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FirstTick uint32 `protobuf:"varint,1,opt,name=first_tick,json=firstTick,proto3" json:"first_tick,omitempty"`
	LastTick  uint32 `protobuf:"varint,2,opt,name=last_tick,json=lastTick,proto3" json:"last_tick,omitempty"`
}

func (x *SyncTickInfoRequest) Reset() {
	*x = SyncTickInfoRequest{}
	mi := &file_sync_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncTickInfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncTickInfoRequest) ProtoMessage() {}

func (x *SyncTickInfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncTickInfoRequest.ProtoReflect.Descriptor instead.
func (*SyncTickInfoRequest) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{5}
}

func (x *SyncTickInfoRequest) GetFirstTick() uint32 {
	if x != nil {
		return x.FirstTick
	}
	return 0
}

func (x *SyncTickInfoRequest) GetLastTick() uint32 {
	if x != nil {
		return x.LastTick
	}
	return 0
}

type SyncTickInfoResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ticks []*SyncTickData `protobuf:"bytes,1,rep,name=ticks,proto3" json:"ticks,omitempty"`
}

func (x *SyncTickInfoResponse) Reset() {
	*x = SyncTickInfoResponse{}
	mi := &file_sync_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncTickInfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncTickInfoResponse) ProtoMessage() {}

func (x *SyncTickInfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncTickInfoResponse.ProtoReflect.Descriptor instead.
func (*SyncTickInfoResponse) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{6}
}

func (x *SyncTickInfoResponse) GetTicks() []*SyncTickData {
	if x != nil {
		return x.Ticks
	}
	return nil
}

type SyncLastSynchronizedTick struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TickNumber uint32 `protobuf:"varint,1,opt,name=tick_number,json=tickNumber,proto3" json:"tick_number,omitempty"`
	Epoch      uint32 `protobuf:"varint,2,opt,name=epoch,proto3" json:"epoch,omitempty"`
	ChainHash  []byte `protobuf:"bytes,3,opt,name=chain_hash,json=chainHash,proto3" json:"chain_hash,omitempty"`
	StoreHash  []byte `protobuf:"bytes,4,opt,name=store_hash,json=storeHash,proto3" json:"store_hash,omitempty"`
}

func (x *SyncLastSynchronizedTick) Reset() {
	*x = SyncLastSynchronizedTick{}
	mi := &file_sync_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SyncLastSynchronizedTick) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncLastSynchronizedTick) ProtoMessage() {}

func (x *SyncLastSynchronizedTick) ProtoReflect() protoreflect.Message {
	mi := &file_sync_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncLastSynchronizedTick.ProtoReflect.Descriptor instead.
func (*SyncLastSynchronizedTick) Descriptor() ([]byte, []int) {
	return file_sync_proto_rawDescGZIP(), []int{7}
}

func (x *SyncLastSynchronizedTick) GetTickNumber() uint32 {
	if x != nil {
		return x.TickNumber
	}
	return 0
}

func (x *SyncLastSynchronizedTick) GetEpoch() uint32 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *SyncLastSynchronizedTick) GetChainHash() []byte {
	if x != nil {
		return x.ChainHash
	}
	return nil
}

func (x *SyncLastSynchronizedTick) GetStoreHash() []byte {
	if x != nil {
		return x.StoreHash
	}
	return nil
}

var File_sync_proto protoreflect.FileDescriptor

var file_sync_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x73, 0x79, 0x6e, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x71, 0x75,
	0x62, 0x69, 0x63, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63,
	0x68, 0x69, 0x76, 0x65, 0x2e, 0x70, 0x62, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0d, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0xe8, 0x01, 0x0a, 0x0d, 0x53, 0x79, 0x6e, 0x63, 0x45, 0x70, 0x6f, 0x63,
	0x68, 0x44, 0x61, 0x74, 0x61, 0x12, 0x49, 0x0a, 0x0d, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x6f,
	0x72, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x71,
	0x75, 0x62, 0x69, 0x63, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72,
	0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x6f,
	0x72, 0x73, 0x52, 0x0c, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x6f, 0x72, 0x4c, 0x69, 0x73, 0x74,
	0x12, 0x8b, 0x01, 0x0a, 0x23, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x5f, 0x71,
	0x75, 0x6f, 0x72, 0x75, 0x6d, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x3e,
	0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e,
	0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x70, 0x62, 0x2e, 0x4c, 0x61, 0x73, 0x74, 0x54,
	0x69, 0x63, 0x6b, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x44, 0x61, 0x74, 0x61, 0x50, 0x65, 0x72,
	0x45, 0x70, 0x6f, 0x63, 0x68, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x73, 0x52, 0x1e,
	0x6c, 0x61, 0x73, 0x74, 0x54, 0x69, 0x63, 0x6b, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x44, 0x61,
	0x74, 0x61, 0x50, 0x65, 0x72, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x73, 0x22, 0xc7,
	0x02, 0x0a, 0x0c, 0x53, 0x79, 0x6e, 0x63, 0x54, 0x69, 0x63, 0x6b, 0x44, 0x61, 0x74, 0x61, 0x12,
	0x40, 0x0a, 0x09, 0x74, 0x69, 0x63, 0x6b, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x23, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69,
	0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x70, 0x62, 0x2e, 0x54,
	0x69, 0x63, 0x6b, 0x44, 0x61, 0x74, 0x61, 0x52, 0x08, 0x74, 0x69, 0x63, 0x6b, 0x44, 0x61, 0x74,
	0x61, 0x12, 0x4a, 0x0a, 0x0b, 0x71, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x5f, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e, 0x61,
	0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e,
	0x70, 0x62, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x54, 0x69, 0x63, 0x6b, 0x44, 0x61, 0x74,
	0x61, 0x52, 0x0a, 0x71, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x44, 0x61, 0x74, 0x61, 0x12, 0x4a, 0x0a,
	0x0c, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x03, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e, 0x61, 0x72, 0x63, 0x68,
	0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x70, 0x62, 0x2e,
	0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x5d, 0x0a, 0x13, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e, 0x61,
	0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e,
	0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x52, 0x12, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0xe2, 0x01, 0x0a, 0x14, 0x53, 0x79, 0x6e,
	0x63, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x29, 0x0a, 0x10, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x5f, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x61, 0x72, 0x63,
	0x68, 0x69, 0x76, 0x65, 0x72, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x2a, 0x0a, 0x10,
	0x6d, 0x61, 0x78, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x10, 0x6d, 0x61, 0x78, 0x4f, 0x62, 0x6a, 0x65, 0x63,
	0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x73, 0x0a, 0x18, 0x70, 0x72, 0x6f, 0x63,
	0x65, 0x73, 0x73, 0x65, 0x64, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x76, 0x61, 0x6c, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x39, 0x2e, 0x71, 0x75, 0x62,
	0x69, 0x63, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68,
	0x69, 0x76, 0x65, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64,
	0x54, 0x69, 0x63, 0x6b, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x73, 0x50, 0x65, 0x72,
	0x45, 0x70, 0x6f, 0x63, 0x68, 0x52, 0x16, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64,
	0x54, 0x69, 0x63, 0x6b, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x73, 0x22, 0x2e, 0x0a,
	0x14, 0x53, 0x79, 0x6e, 0x63, 0x45, 0x70, 0x6f, 0x63, 0x68, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x06, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x73, 0x22, 0x59, 0x0a,
	0x15, 0x53, 0x79, 0x6e, 0x63, 0x45, 0x70, 0x6f, 0x63, 0x68, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x40, 0x0a, 0x06, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e, 0x61,
	0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e,
	0x70, 0x62, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x45, 0x70, 0x6f, 0x63, 0x68, 0x44, 0x61, 0x74, 0x61,
	0x52, 0x06, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x73, 0x22, 0x51, 0x0a, 0x13, 0x53, 0x79, 0x6e, 0x63,
	0x54, 0x69, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x1d, 0x0a, 0x0a, 0x66, 0x69, 0x72, 0x73, 0x74, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x09, 0x66, 0x69, 0x72, 0x73, 0x74, 0x54, 0x69, 0x63, 0x6b, 0x12, 0x1b,
	0x0a, 0x09, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x08, 0x6c, 0x61, 0x73, 0x74, 0x54, 0x69, 0x63, 0x6b, 0x22, 0x55, 0x0a, 0x14, 0x53,
	0x79, 0x6e, 0x63, 0x54, 0x69, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x3d, 0x0a, 0x05, 0x74, 0x69, 0x63, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x27, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69,
	0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x70, 0x62, 0x2e, 0x53,
	0x79, 0x6e, 0x63, 0x54, 0x69, 0x63, 0x6b, 0x44, 0x61, 0x74, 0x61, 0x52, 0x05, 0x74, 0x69, 0x63,
	0x6b, 0x73, 0x22, 0x8f, 0x01, 0x0a, 0x18, 0x53, 0x79, 0x6e, 0x63, 0x4c, 0x61, 0x73, 0x74, 0x53,
	0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x65, 0x64, 0x54, 0x69, 0x63, 0x6b, 0x12,
	0x1f, 0x0a, 0x0b, 0x74, 0x69, 0x63, 0x6b, 0x5f, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x74, 0x69, 0x63, 0x6b, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72,
	0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x5f,
	0x68, 0x61, 0x73, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x48, 0x61, 0x73, 0x68, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x5f, 0x68,
	0x61, 0x73, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x74, 0x6f, 0x72, 0x65,
	0x48, 0x61, 0x73, 0x68, 0x32, 0xf6, 0x02, 0x0a, 0x0b, 0x53, 0x79, 0x6e, 0x63, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x65, 0x0a, 0x18, 0x53, 0x79, 0x6e, 0x63, 0x47, 0x65, 0x74, 0x42,
	0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x2f, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63,
	0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76,
	0x65, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x80, 0x01, 0x0a, 0x17,
	0x53, 0x79, 0x6e, 0x63, 0x47, 0x65, 0x74, 0x45, 0x70, 0x6f, 0x63, 0x68, 0x49, 0x6e, 0x66, 0x6f,
	0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2f, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63, 0x2e,
	0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65,
	0x2e, 0x70, 0x62, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x45, 0x70, 0x6f, 0x63, 0x68, 0x49, 0x6e, 0x66,
	0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x30, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63,
	0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76,
	0x65, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x45, 0x70, 0x6f, 0x63, 0x68, 0x49, 0x6e,
	0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x30, 0x01, 0x12, 0x7d,
	0x0a, 0x16, 0x53, 0x79, 0x6e, 0x63, 0x47, 0x65, 0x74, 0x54, 0x69, 0x63, 0x6b, 0x49, 0x6e, 0x66,
	0x6f, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2e, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63,
	0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76,
	0x65, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x54, 0x69, 0x63, 0x6b, 0x49, 0x6e, 0x66,
	0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2f, 0x2e, 0x71, 0x75, 0x62, 0x69, 0x63,
	0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76,
	0x65, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x54, 0x69, 0x63, 0x6b, 0x49, 0x6e, 0x66,
	0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x30, 0x01, 0x42, 0x29, 0x5a,
	0x27, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x71, 0x75, 0x62, 0x69,
	0x63, 0x2f, 0x67, 0x6f, 0x2d, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x72, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x66, 0x2f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_sync_proto_rawDescOnce sync.Once
	file_sync_proto_rawDescData = file_sync_proto_rawDesc
)

func file_sync_proto_rawDescGZIP() []byte {
	file_sync_proto_rawDescOnce.Do(func() {
		file_sync_proto_rawDescData = protoimpl.X.CompressGZIP(file_sync_proto_rawDescData)
	})
	return file_sync_proto_rawDescData
}

var file_sync_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_sync_proto_goTypes = []any{
	(*SyncEpochData)(nil),                       // 0: qubic.archiver.archive.pb.SyncEpochData
	(*SyncTickData)(nil),                        // 1: qubic.archiver.archive.pb.SyncTickData
	(*SyncMetadataResponse)(nil),                // 2: qubic.archiver.archive.pb.SyncMetadataResponse
	(*SyncEpochInfoRequest)(nil),                // 3: qubic.archiver.archive.pb.SyncEpochInfoRequest
	(*SyncEpochInfoResponse)(nil),               // 4: qubic.archiver.archive.pb.SyncEpochInfoResponse
	(*SyncTickInfoRequest)(nil),                 // 5: qubic.archiver.archive.pb.SyncTickInfoRequest
	(*SyncTickInfoResponse)(nil),                // 6: qubic.archiver.archive.pb.SyncTickInfoResponse
	(*SyncLastSynchronizedTick)(nil),            // 7: qubic.archiver.archive.pb.SyncLastSynchronizedTick
	(*Computors)(nil),                           // 8: qubic.archiver.archive.pb.Computors
	(*LastTickQuorumDataPerEpochIntervals)(nil), // 9: qubic.archiver.archive.pb.LastTickQuorumDataPerEpochIntervals
	(*TickData)(nil),                            // 10: qubic.archiver.archive.pb.TickData
	(*QuorumTickData)(nil),                      // 11: qubic.archiver.archive.pb.QuorumTickData
	(*Transaction)(nil),                         // 12: qubic.archiver.archive.pb.Transaction
	(*TransactionStatus)(nil),                   // 13: qubic.archiver.archive.pb.TransactionStatus
	(*ProcessedTickIntervalsPerEpoch)(nil),      // 14: qubic.archiver.archive.pb.ProcessedTickIntervalsPerEpoch
	(*emptypb.Empty)(nil),                       // 15: google.protobuf.Empty
}
var file_sync_proto_depIdxs = []int32{
	8,  // 0: qubic.archiver.archive.pb.SyncEpochData.computor_list:type_name -> qubic.archiver.archive.pb.Computors
	9,  // 1: qubic.archiver.archive.pb.SyncEpochData.last_tick_quorum_data_per_intervals:type_name -> qubic.archiver.archive.pb.LastTickQuorumDataPerEpochIntervals
	10, // 2: qubic.archiver.archive.pb.SyncTickData.tick_data:type_name -> qubic.archiver.archive.pb.TickData
	11, // 3: qubic.archiver.archive.pb.SyncTickData.quorum_data:type_name -> qubic.archiver.archive.pb.QuorumTickData
	12, // 4: qubic.archiver.archive.pb.SyncTickData.transactions:type_name -> qubic.archiver.archive.pb.Transaction
	13, // 5: qubic.archiver.archive.pb.SyncTickData.transactions_status:type_name -> qubic.archiver.archive.pb.TransactionStatus
	14, // 6: qubic.archiver.archive.pb.SyncMetadataResponse.processed_tick_intervals:type_name -> qubic.archiver.archive.pb.ProcessedTickIntervalsPerEpoch
	0,  // 7: qubic.archiver.archive.pb.SyncEpochInfoResponse.epochs:type_name -> qubic.archiver.archive.pb.SyncEpochData
	1,  // 8: qubic.archiver.archive.pb.SyncTickInfoResponse.ticks:type_name -> qubic.archiver.archive.pb.SyncTickData
	15, // 9: qubic.archiver.archive.pb.SyncService.SyncGetBootstrapMetadata:input_type -> google.protobuf.Empty
	3,  // 10: qubic.archiver.archive.pb.SyncService.SyncGetEpochInformation:input_type -> qubic.archiver.archive.pb.SyncEpochInfoRequest
	5,  // 11: qubic.archiver.archive.pb.SyncService.SyncGetTickInformation:input_type -> qubic.archiver.archive.pb.SyncTickInfoRequest
	2,  // 12: qubic.archiver.archive.pb.SyncService.SyncGetBootstrapMetadata:output_type -> qubic.archiver.archive.pb.SyncMetadataResponse
	4,  // 13: qubic.archiver.archive.pb.SyncService.SyncGetEpochInformation:output_type -> qubic.archiver.archive.pb.SyncEpochInfoResponse
	6,  // 14: qubic.archiver.archive.pb.SyncService.SyncGetTickInformation:output_type -> qubic.archiver.archive.pb.SyncTickInfoResponse
	12, // [12:15] is the sub-list for method output_type
	9,  // [9:12] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_sync_proto_init() }
func file_sync_proto_init() {
	if File_sync_proto != nil {
		return
	}
	file_archive_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_sync_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_sync_proto_goTypes,
		DependencyIndexes: file_sync_proto_depIdxs,
		MessageInfos:      file_sync_proto_msgTypes,
	}.Build()
	File_sync_proto = out.File
	file_sync_proto_rawDesc = nil
	file_sync_proto_goTypes = nil
	file_sync_proto_depIdxs = nil
}
