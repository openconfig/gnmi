# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# NO CHECKED-IN PROTOBUF GENCODE
# source: testing/fake/proto/fake.proto
# Protobuf Python Version: 5.27.2
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(
    _runtime_version.Domain.PUBLIC,
    5,
    27,
    2,
    '',
    'testing/fake/proto/fake.proto'
)
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from google.protobuf import any_pb2 as google_dot_protobuf_dot_any__pb2
from github.com.openconfig.gnmi.proto.gnmi import gnmi_pb2 as github_dot_com_dot_openconfig_dot_gnmi_dot_proto_dot_gnmi_dot_gnmi__pb2


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x1dtesting/fake/proto/fake.proto\x12\tgnmi.fake\x1a\x19google/protobuf/any.proto\x1a\x30github.com/openconfig/gnmi/proto/gnmi/gnmi.proto\"2\n\rConfiguration\x12!\n\x06\x63onfig\x18\x01 \x03(\x0b\x32\x11.gnmi.fake.Config\"1\n\x0b\x43redentials\x12\x10\n\x08username\x18\x01 \x01(\t\x12\x10\n\x08password\x18\x02 \x01(\t\"\x8c\x04\n\x06\x43onfig\x12\x0e\n\x06target\x18\x01 \x01(\t\x12\x0c\n\x04port\x18\x02 \x01(\x05\x12\x10\n\x04seed\x18\x06 \x01(\x03\x42\x02\x18\x01\x12$\n\x06values\x18\x03 \x03(\x0b\x32\x10.gnmi.fake.ValueB\x02\x18\x01\x12\x14\n\x0c\x64isable_sync\x18\x04 \x01(\x08\x12\x31\n\x0b\x63lient_type\x18\x05 \x01(\x0e\x32\x1c.gnmi.fake.Config.ClientType\x12\x13\n\x0b\x64isable_eof\x18\x07 \x01(\x08\x12+\n\x0b\x63redentials\x18\x08 \x01(\x0b\x32\x16.gnmi.fake.Credentials\x12\x0c\n\x04\x63\x65rt\x18\t \x01(\x0c\x12\x14\n\x0c\x65nable_delay\x18\n \x01(\x08\x12&\n\x06\x63ustom\x18\x64 \x01(\x0b\x32\x14.google.protobuf.AnyH\x00\x12,\n\x06random\x18\x65 \x01(\x0b\x32\x1a.gnmi.fake.RandomGeneratorH\x00\x12*\n\x05\x66ixed\x18\x66 \x01(\x0b\x32\x19.gnmi.fake.FixedGeneratorH\x00\x12\x13\n\x0btunnel_addr\x18\x0b \x01(\t\x12\x12\n\ntunnel_crt\x18\x0c \x01(\t\"E\n\nClientType\x12\x08\n\x04GRPC\x10\x00\x12\n\n\x06STUBBY\x10\x01\x12\r\n\tGRPC_GNMI\x10\x02\x12\x12\n\x0eGRPC_GNMI_PROD\x10\x03\x42\x0b\n\tgenerator\"<\n\x0e\x46ixedGenerator\x12*\n\tresponses\x18\x01 \x03(\x0b\x32\x17.gnmi.SubscribeResponse\"A\n\x0fRandomGenerator\x12\x0c\n\x04seed\x18\x01 \x01(\x03\x12 \n\x06values\x18\x02 \x03(\x0b\x32\x10.gnmi.fake.Value\"\r\n\x0b\x44\x65leteValue\"\xba\x03\n\x05Value\x12\x0c\n\x04path\x18\x01 \x03(\t\x12\'\n\ttimestamp\x18\x02 \x01(\x0b\x32\x14.gnmi.fake.Timestamp\x12\x0e\n\x06repeat\x18\x06 \x01(\x05\x12\x0c\n\x04seed\x18\x07 \x01(\x03\x12(\n\tint_value\x18\x64 \x01(\x0b\x32\x13.gnmi.fake.IntValueH\x00\x12.\n\x0c\x64ouble_value\x18\x65 \x01(\x0b\x32\x16.gnmi.fake.DoubleValueH\x00\x12.\n\x0cstring_value\x18\x66 \x01(\x0b\x32\x16.gnmi.fake.StringValueH\x00\x12\x0e\n\x04sync\x18g \x01(\x04H\x00\x12(\n\x06\x64\x65lete\x18h \x01(\x0b\x32\x16.gnmi.fake.DeleteValueH\x00\x12*\n\nbool_value\x18i \x01(\x0b\x32\x14.gnmi.fake.BoolValueH\x00\x12*\n\nuint_value\x18j \x01(\x0b\x32\x14.gnmi.fake.UintValueH\x00\x12\x37\n\x11string_list_value\x18k \x01(\x0b\x32\x1a.gnmi.fake.StringListValueH\x00\x42\x07\n\x05value\"D\n\tTimestamp\x12\x11\n\ttimestamp\x18\x01 \x01(\x03\x12\x11\n\tdelta_min\x18\x02 \x01(\x03\x12\x11\n\tdelta_max\x18\x03 \x01(\x03\"s\n\x08IntValue\x12\r\n\x05value\x18\x01 \x01(\x03\x12$\n\x05range\x18\x02 \x01(\x0b\x32\x13.gnmi.fake.IntRangeH\x00\x12\"\n\x04list\x18\x03 \x01(\x0b\x32\x12.gnmi.fake.IntListH\x00\x42\x0e\n\x0c\x64istribution\"R\n\x08IntRange\x12\x0f\n\x07minimum\x18\x01 \x01(\x03\x12\x0f\n\x07maximum\x18\x02 \x01(\x03\x12\x11\n\tdelta_min\x18\x03 \x01(\x03\x12\x11\n\tdelta_max\x18\x04 \x01(\x03\"*\n\x07IntList\x12\x0f\n\x07options\x18\x01 \x03(\x03\x12\x0e\n\x06random\x18\x02 \x01(\x08\"|\n\x0b\x44oubleValue\x12\r\n\x05value\x18\x01 \x01(\x01\x12\'\n\x05range\x18\x02 \x01(\x0b\x32\x16.gnmi.fake.DoubleRangeH\x00\x12%\n\x04list\x18\x03 \x01(\x0b\x32\x15.gnmi.fake.DoubleListH\x00\x42\x0e\n\x0c\x64istribution\"U\n\x0b\x44oubleRange\x12\x0f\n\x07minimum\x18\x01 \x01(\x01\x12\x0f\n\x07maximum\x18\x02 \x01(\x01\x12\x11\n\tdelta_min\x18\x03 \x01(\x01\x12\x11\n\tdelta_max\x18\x04 \x01(\x01\"-\n\nDoubleList\x12\x0f\n\x07options\x18\x01 \x03(\x01\x12\x0e\n\x06random\x18\x02 \x01(\x08\"S\n\x0bStringValue\x12\r\n\x05value\x18\x01 \x01(\t\x12%\n\x04list\x18\x02 \x01(\x0b\x32\x15.gnmi.fake.StringListH\x00\x42\x0e\n\x0c\x64istribution\"-\n\nStringList\x12\x0f\n\x07options\x18\x01 \x03(\t\x12\x0e\n\x06random\x18\x02 \x01(\x08\"W\n\x0fStringListValue\x12\r\n\x05value\x18\x01 \x03(\t\x12%\n\x04list\x18\x02 \x01(\x0b\x32\x15.gnmi.fake.StringListH\x00\x42\x0e\n\x0c\x64istribution\"O\n\tBoolValue\x12\r\n\x05value\x18\x01 \x01(\x08\x12#\n\x04list\x18\x02 \x01(\x0b\x32\x13.gnmi.fake.BoolListH\x00\x42\x0e\n\x0c\x64istribution\"+\n\x08\x42oolList\x12\x0f\n\x07options\x18\x01 \x03(\x08\x12\x0e\n\x06random\x18\x02 \x01(\x08\"v\n\tUintValue\x12\r\n\x05value\x18\x01 \x01(\x04\x12%\n\x05range\x18\x02 \x01(\x0b\x32\x14.gnmi.fake.UintRangeH\x00\x12#\n\x04list\x18\x03 \x01(\x0b\x32\x13.gnmi.fake.UintListH\x00\x42\x0e\n\x0c\x64istribution\"S\n\tUintRange\x12\x0f\n\x07minimum\x18\x01 \x01(\x04\x12\x0f\n\x07maximum\x18\x02 \x01(\x04\x12\x11\n\tdelta_min\x18\x03 \x01(\x03\x12\x11\n\tdelta_max\x18\x04 \x01(\x03\"+\n\x08UintList\x12\x0f\n\x07options\x18\x01 \x03(\x04\x12\x0e\n\x06random\x18\x02 \x01(\x08*+\n\x05State\x12\x0b\n\x07STOPPED\x10\x00\x12\x08\n\x04INIT\x10\x01\x12\x0b\n\x07RUNNING\x10\x02\x32\x9b\x01\n\x0c\x41gentManager\x12+\n\x03\x41\x64\x64\x12\x11.gnmi.fake.Config\x1a\x11.gnmi.fake.Config\x12.\n\x06Remove\x12\x11.gnmi.fake.Config\x1a\x11.gnmi.fake.Config\x12.\n\x06Status\x12\x11.gnmi.fake.Config\x1a\x11.gnmi.fake.ConfigB9Z7github.com/openconfig/gnmi/testing/fake/proto;gnmi_fakeb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'testing.fake.proto.fake_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z7github.com/openconfig/gnmi/testing/fake/proto;gnmi_fake'
  _globals['_CONFIG'].fields_by_name['seed']._loaded_options = None
  _globals['_CONFIG'].fields_by_name['seed']._serialized_options = b'\030\001'
  _globals['_CONFIG'].fields_by_name['values']._loaded_options = None
  _globals['_CONFIG'].fields_by_name['values']._serialized_options = b'\030\001'
  _globals['_STATE']._serialized_start=2512
  _globals['_STATE']._serialized_end=2555
  _globals['_CONFIGURATION']._serialized_start=121
  _globals['_CONFIGURATION']._serialized_end=171
  _globals['_CREDENTIALS']._serialized_start=173
  _globals['_CREDENTIALS']._serialized_end=222
  _globals['_CONFIG']._serialized_start=225
  _globals['_CONFIG']._serialized_end=749
  _globals['_CONFIG_CLIENTTYPE']._serialized_start=667
  _globals['_CONFIG_CLIENTTYPE']._serialized_end=736
  _globals['_FIXEDGENERATOR']._serialized_start=751
  _globals['_FIXEDGENERATOR']._serialized_end=811
  _globals['_RANDOMGENERATOR']._serialized_start=813
  _globals['_RANDOMGENERATOR']._serialized_end=878
  _globals['_DELETEVALUE']._serialized_start=880
  _globals['_DELETEVALUE']._serialized_end=893
  _globals['_VALUE']._serialized_start=896
  _globals['_VALUE']._serialized_end=1338
  _globals['_TIMESTAMP']._serialized_start=1340
  _globals['_TIMESTAMP']._serialized_end=1408
  _globals['_INTVALUE']._serialized_start=1410
  _globals['_INTVALUE']._serialized_end=1525
  _globals['_INTRANGE']._serialized_start=1527
  _globals['_INTRANGE']._serialized_end=1609
  _globals['_INTLIST']._serialized_start=1611
  _globals['_INTLIST']._serialized_end=1653
  _globals['_DOUBLEVALUE']._serialized_start=1655
  _globals['_DOUBLEVALUE']._serialized_end=1779
  _globals['_DOUBLERANGE']._serialized_start=1781
  _globals['_DOUBLERANGE']._serialized_end=1866
  _globals['_DOUBLELIST']._serialized_start=1868
  _globals['_DOUBLELIST']._serialized_end=1913
  _globals['_STRINGVALUE']._serialized_start=1915
  _globals['_STRINGVALUE']._serialized_end=1998
  _globals['_STRINGLIST']._serialized_start=2000
  _globals['_STRINGLIST']._serialized_end=2045
  _globals['_STRINGLISTVALUE']._serialized_start=2047
  _globals['_STRINGLISTVALUE']._serialized_end=2134
  _globals['_BOOLVALUE']._serialized_start=2136
  _globals['_BOOLVALUE']._serialized_end=2215
  _globals['_BOOLLIST']._serialized_start=2217
  _globals['_BOOLLIST']._serialized_end=2260
  _globals['_UINTVALUE']._serialized_start=2262
  _globals['_UINTVALUE']._serialized_end=2380
  _globals['_UINTRANGE']._serialized_start=2382
  _globals['_UINTRANGE']._serialized_end=2465
  _globals['_UINTLIST']._serialized_start=2467
  _globals['_UINTLIST']._serialized_end=2510
  _globals['_AGENTMANAGER']._serialized_start=2558
  _globals['_AGENTMANAGER']._serialized_end=2713
# @@protoc_insertion_point(module_scope)
