// Code generated by protoc-gen-go. DO NOT EDIT.
// source: discovery/discoverypb/discovery.proto

/*
Package discoverypb is a generated protocol buffer package.

It is generated from these files:
	discovery/discoverypb/discovery.proto

It has these top-level messages:
	RegisterRequest
	RegisterResponse
	DeregisterRequest
	DeregisterResponse
*/
package discoverypb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type RegisterRequest struct {
	Directory string `protobuf:"bytes,1,opt,name=directory" json:"directory,omitempty"`
	Address   string `protobuf:"bytes,2,opt,name=address" json:"address,omitempty"`
}

func (m *RegisterRequest) Reset()                    { *m = RegisterRequest{} }
func (m *RegisterRequest) String() string            { return proto.CompactTextString(m) }
func (*RegisterRequest) ProtoMessage()               {}
func (*RegisterRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *RegisterRequest) GetDirectory() string {
	if m != nil {
		return m.Directory
	}
	return ""
}

func (m *RegisterRequest) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type RegisterResponse struct {
}

func (m *RegisterResponse) Reset()                    { *m = RegisterResponse{} }
func (m *RegisterResponse) String() string            { return proto.CompactTextString(m) }
func (*RegisterResponse) ProtoMessage()               {}
func (*RegisterResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type DeregisterRequest struct {
	Directory string `protobuf:"bytes,1,opt,name=directory" json:"directory,omitempty"`
}

func (m *DeregisterRequest) Reset()                    { *m = DeregisterRequest{} }
func (m *DeregisterRequest) String() string            { return proto.CompactTextString(m) }
func (*DeregisterRequest) ProtoMessage()               {}
func (*DeregisterRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *DeregisterRequest) GetDirectory() string {
	if m != nil {
		return m.Directory
	}
	return ""
}

type DeregisterResponse struct {
}

func (m *DeregisterResponse) Reset()                    { *m = DeregisterResponse{} }
func (m *DeregisterResponse) String() string            { return proto.CompactTextString(m) }
func (*DeregisterResponse) ProtoMessage()               {}
func (*DeregisterResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func init() {
	proto.RegisterType((*RegisterRequest)(nil), "discoverypb.RegisterRequest")
	proto.RegisterType((*RegisterResponse)(nil), "discoverypb.RegisterResponse")
	proto.RegisterType((*DeregisterRequest)(nil), "discoverypb.DeregisterRequest")
	proto.RegisterType((*DeregisterResponse)(nil), "discoverypb.DeregisterResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for DiscoveryService service

type DiscoveryServiceClient interface {
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	Deregister(ctx context.Context, in *DeregisterRequest, opts ...grpc.CallOption) (*DeregisterResponse, error)
}

type discoveryServiceClient struct {
	cc *grpc.ClientConn
}

func NewDiscoveryServiceClient(cc *grpc.ClientConn) DiscoveryServiceClient {
	return &discoveryServiceClient{cc}
}

func (c *discoveryServiceClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := grpc.Invoke(ctx, "/discoverypb.DiscoveryService/Register", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *discoveryServiceClient) Deregister(ctx context.Context, in *DeregisterRequest, opts ...grpc.CallOption) (*DeregisterResponse, error) {
	out := new(DeregisterResponse)
	err := grpc.Invoke(ctx, "/discoverypb.DiscoveryService/Deregister", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for DiscoveryService service

type DiscoveryServiceServer interface {
	Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
	Deregister(context.Context, *DeregisterRequest) (*DeregisterResponse, error)
}

func RegisterDiscoveryServiceServer(s *grpc.Server, srv DiscoveryServiceServer) {
	s.RegisterService(&_DiscoveryService_serviceDesc, srv)
}

func _DiscoveryService_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiscoveryServiceServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/discoverypb.DiscoveryService/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiscoveryServiceServer).Register(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DiscoveryService_Deregister_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeregisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiscoveryServiceServer).Deregister(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/discoverypb.DiscoveryService/Deregister",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiscoveryServiceServer).Deregister(ctx, req.(*DeregisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _DiscoveryService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "discoverypb.DiscoveryService",
	HandlerType: (*DiscoveryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Register",
			Handler:    _DiscoveryService_Register_Handler,
		},
		{
			MethodName: "Deregister",
			Handler:    _DiscoveryService_Deregister_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "discovery/discoverypb/discovery.proto",
}

func init() { proto.RegisterFile("discovery/discoverypb/discovery.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 201 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x4d, 0xc9, 0x2c, 0x4e,
	0xce, 0x2f, 0x4b, 0x2d, 0xaa, 0xd4, 0x87, 0xb3, 0x0a, 0x92, 0x10, 0x6c, 0xbd, 0x82, 0xa2, 0xfc,
	0x92, 0x7c, 0x21, 0x6e, 0x24, 0x49, 0x25, 0x4f, 0x2e, 0xfe, 0xa0, 0xd4, 0xf4, 0xcc, 0xe2, 0x92,
	0xd4, 0xa2, 0xa0, 0xd4, 0xc2, 0xd2, 0xd4, 0xe2, 0x12, 0x21, 0x19, 0x2e, 0xce, 0x94, 0xcc, 0xa2,
	0xd4, 0xe4, 0x92, 0xfc, 0xa2, 0x4a, 0x09, 0x46, 0x05, 0x46, 0x0d, 0xce, 0x20, 0x84, 0x80, 0x90,
	0x04, 0x17, 0x7b, 0x62, 0x4a, 0x4a, 0x51, 0x6a, 0x71, 0xb1, 0x04, 0x13, 0x58, 0x0e, 0xc6, 0x55,
	0x12, 0xe2, 0x12, 0x40, 0x18, 0x55, 0x5c, 0x90, 0x9f, 0x57, 0x9c, 0xaa, 0x64, 0xc8, 0x25, 0xe8,
	0x92, 0x5a, 0x44, 0x8a, 0x05, 0x4a, 0x22, 0x5c, 0x42, 0xc8, 0x5a, 0x20, 0x06, 0x19, 0xad, 0x63,
	0xe4, 0x12, 0x70, 0x81, 0xb9, 0x3b, 0x38, 0xb5, 0xa8, 0x2c, 0x33, 0x39, 0x55, 0xc8, 0x93, 0x8b,
	0x03, 0x66, 0xa3, 0x90, 0x8c, 0x1e, 0x92, 0xb7, 0xf4, 0xd0, 0xfc, 0x24, 0x25, 0x8b, 0x43, 0x16,
	0xea, 0x4c, 0x06, 0x21, 0x7f, 0x2e, 0x2e, 0x84, 0xad, 0x42, 0x72, 0x28, 0xca, 0x31, 0x7c, 0x20,
	0x25, 0x8f, 0x53, 0x1e, 0x66, 0x60, 0x12, 0x1b, 0x38, 0xb0, 0x8d, 0x01, 0x01, 0x00, 0x00, 0xff,
	0xff, 0xd2, 0x74, 0xfa, 0xbb, 0x95, 0x01, 0x00, 0x00,
}
