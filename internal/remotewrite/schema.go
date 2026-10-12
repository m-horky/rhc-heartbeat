package remotewrite

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FIXME This should get generated from a .proto file instead.

// remoteWriteDescriptors hold message structure definition
// for the Prometheus remote write v1 protocol schema.
type remoteWriteDescriptors struct {
	writeRequest protoreflect.MessageDescriptor
	timeSeries   protoreflect.MessageDescriptor
	label        protoreflect.MessageDescriptor
	sample       protoreflect.MessageDescriptor
}

// remoteWriteFileDescriptor defines the Prometheus remote write 1 schema.
var remoteWriteFileDescriptor = &descriptorpb.FileDescriptorProto{ //nolint:gochecknoglobals
	Name:    new("prometheus/write_request.proto"),
	Package: new("prometheus"),
	Syntax:  new("proto3"),
	MessageType: []*descriptorpb.DescriptorProto{
		{
			Name: new("WriteRequest"),
			ReservedRange: []*descriptorpb.DescriptorProto_ReservedRange{
				{Start: proto.Int32(2), End: proto.Int32(4)},
			},
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     new("timeseries"),
					Number:   new(int32(1)),
					Label:    new(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
					TypeName: new(".prometheus.TimeSeries"),
				},
			},
		},
		{
			Name: new("TimeSeries"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     new("labels"),
					Number:   new(int32(1)),
					Label:    new(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
					TypeName: new(".prometheus.Label"),
				},
				{
					Name:     new("samples"),
					Number:   new(int32(2)),
					Label:    new(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
					TypeName: new(".prometheus.Sample"),
				},
			},
		},
		{
			Name: new("Label"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:   new("name"),
					Number: new(int32(1)),
					Label:  new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					Type:   new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
				},
				{
					Name:   new("value"),
					Number: new(int32(2)),
					Label:  new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					Type:   new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
				},
			},
		},
		{
			Name: new("Sample"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:   new("value"),
					Number: new(int32(1)),
					Label:  new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					Type:   new(descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
				},
				{
					Name:   new("timestamp"),
					Number: new(int32(2)),
					Label:  new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					Type:   new(descriptorpb.FieldDescriptorProto_TYPE_INT64),
				},
			},
		},
	},
}
