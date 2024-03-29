# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: proto/gnmi_ext/gnmi_ext.proto

import sys
_b=sys.version_info[0]<3 and (lambda x:x) or (lambda x:x.encode('latin1'))
from google.protobuf.internal import enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
from google.protobuf import descriptor_pb2
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from google.protobuf import duration_pb2 as google_dot_protobuf_dot_duration__pb2


DESCRIPTOR = _descriptor.FileDescriptor(
  name='proto/gnmi_ext/gnmi_ext.proto',
  package='gnmi_ext',
  syntax='proto3',
  serialized_pb=_b('\n\x1dproto/gnmi_ext/gnmi_ext.proto\x12\x08gnmi_ext\x1a\x1egoogle/protobuf/duration.proto\"\xf2\x01\n\tExtension\x12\x37\n\x0eregistered_ext\x18\x01 \x01(\x0b\x32\x1d.gnmi_ext.RegisteredExtensionH\x00\x12\x39\n\x12master_arbitration\x18\x02 \x01(\x0b\x32\x1b.gnmi_ext.MasterArbitrationH\x00\x12$\n\x07history\x18\x03 \x01(\x0b\x32\x11.gnmi_ext.HistoryH\x00\x12\"\n\x06\x63ommit\x18\x04 \x01(\x0b\x32\x10.gnmi_ext.CommitH\x00\x12 \n\x05\x64\x65pth\x18\x05 \x01(\x0b\x32\x0f.gnmi_ext.DepthH\x00\x42\x05\n\x03\x65xt\"E\n\x13RegisteredExtension\x12!\n\x02id\x18\x01 \x01(\x0e\x32\x15.gnmi_ext.ExtensionID\x12\x0b\n\x03msg\x18\x02 \x01(\x0c\"Y\n\x11MasterArbitration\x12\x1c\n\x04role\x18\x01 \x01(\x0b\x32\x0e.gnmi_ext.Role\x12&\n\x0b\x65lection_id\x18\x02 \x01(\x0b\x32\x11.gnmi_ext.Uint128\"$\n\x07Uint128\x12\x0c\n\x04high\x18\x01 \x01(\x04\x12\x0b\n\x03low\x18\x02 \x01(\x04\"\x12\n\x04Role\x12\n\n\x02id\x18\x01 \x01(\t\"S\n\x07History\x12\x17\n\rsnapshot_time\x18\x01 \x01(\x03H\x00\x12$\n\x05range\x18\x02 \x01(\x0b\x32\x13.gnmi_ext.TimeRangeH\x00\x42\t\n\x07request\"\'\n\tTimeRange\x12\r\n\x05start\x18\x01 \x01(\x03\x12\x0b\n\x03\x65nd\x18\x02 \x01(\x03\"\xe5\x01\n\x06\x43ommit\x12\n\n\x02id\x18\x01 \x01(\t\x12)\n\x06\x63ommit\x18\x02 \x01(\x0b\x32\x17.gnmi_ext.CommitRequestH\x00\x12*\n\x07\x63onfirm\x18\x03 \x01(\x0b\x32\x17.gnmi_ext.CommitConfirmH\x00\x12(\n\x06\x63\x61ncel\x18\x04 \x01(\x0b\x32\x16.gnmi_ext.CommitCancelH\x00\x12\x44\n\x15set_rollback_duration\x18\x05 \x01(\x0b\x32#.gnmi_ext.CommitSetRollbackDurationH\x00\x42\x08\n\x06\x61\x63tion\"E\n\rCommitRequest\x12\x34\n\x11rollback_duration\x18\x01 \x01(\x0b\x32\x19.google.protobuf.Duration\"\x0f\n\rCommitConfirm\"\x0e\n\x0c\x43ommitCancel\"Q\n\x19\x43ommitSetRollbackDuration\x12\x34\n\x11rollback_duration\x18\x01 \x01(\x0b\x32\x19.google.protobuf.Duration\"\x16\n\x05\x44\x65pth\x12\r\n\x05level\x18\x01 \x01(\r*3\n\x0b\x45xtensionID\x12\r\n\tEID_UNSET\x10\x00\x12\x15\n\x10\x45ID_EXPERIMENTAL\x10\xe7\x07\x42+Z)github.com/openconfig/gnmi/proto/gnmi_extb\x06proto3')
  ,
  dependencies=[google_dot_protobuf_dot_duration__pb2.DESCRIPTOR,])

_EXTENSIONID = _descriptor.EnumDescriptor(
  name='ExtensionID',
  full_name='gnmi_ext.ExtensionID',
  filename=None,
  file=DESCRIPTOR,
  values=[
    _descriptor.EnumValueDescriptor(
      name='EID_UNSET', index=0, number=0,
      options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='EID_EXPERIMENTAL', index=1, number=999,
      options=None,
      type=None),
  ],
  containing_type=None,
  options=None,
  serialized_start=1109,
  serialized_end=1160,
)
_sym_db.RegisterEnumDescriptor(_EXTENSIONID)

ExtensionID = enum_type_wrapper.EnumTypeWrapper(_EXTENSIONID)
EID_UNSET = 0
EID_EXPERIMENTAL = 999



_EXTENSION = _descriptor.Descriptor(
  name='Extension',
  full_name='gnmi_ext.Extension',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='registered_ext', full_name='gnmi_ext.Extension.registered_ext', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='master_arbitration', full_name='gnmi_ext.Extension.master_arbitration', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='history', full_name='gnmi_ext.Extension.history', index=2,
      number=3, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='commit', full_name='gnmi_ext.Extension.commit', index=3,
      number=4, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='depth', full_name='gnmi_ext.Extension.depth', index=4,
      number=5, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
    _descriptor.OneofDescriptor(
      name='ext', full_name='gnmi_ext.Extension.ext',
      index=0, containing_type=None, fields=[]),
  ],
  serialized_start=76,
  serialized_end=318,
)


_REGISTEREDEXTENSION = _descriptor.Descriptor(
  name='RegisteredExtension',
  full_name='gnmi_ext.RegisteredExtension',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='id', full_name='gnmi_ext.RegisteredExtension.id', index=0,
      number=1, type=14, cpp_type=8, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='msg', full_name='gnmi_ext.RegisteredExtension.msg', index=1,
      number=2, type=12, cpp_type=9, label=1,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=320,
  serialized_end=389,
)


_MASTERARBITRATION = _descriptor.Descriptor(
  name='MasterArbitration',
  full_name='gnmi_ext.MasterArbitration',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='role', full_name='gnmi_ext.MasterArbitration.role', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='election_id', full_name='gnmi_ext.MasterArbitration.election_id', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=391,
  serialized_end=480,
)


_UINT128 = _descriptor.Descriptor(
  name='Uint128',
  full_name='gnmi_ext.Uint128',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='high', full_name='gnmi_ext.Uint128.high', index=0,
      number=1, type=4, cpp_type=4, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='low', full_name='gnmi_ext.Uint128.low', index=1,
      number=2, type=4, cpp_type=4, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=482,
  serialized_end=518,
)


_ROLE = _descriptor.Descriptor(
  name='Role',
  full_name='gnmi_ext.Role',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='id', full_name='gnmi_ext.Role.id', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=520,
  serialized_end=538,
)


_HISTORY = _descriptor.Descriptor(
  name='History',
  full_name='gnmi_ext.History',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='snapshot_time', full_name='gnmi_ext.History.snapshot_time', index=0,
      number=1, type=3, cpp_type=2, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='range', full_name='gnmi_ext.History.range', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
    _descriptor.OneofDescriptor(
      name='request', full_name='gnmi_ext.History.request',
      index=0, containing_type=None, fields=[]),
  ],
  serialized_start=540,
  serialized_end=623,
)


_TIMERANGE = _descriptor.Descriptor(
  name='TimeRange',
  full_name='gnmi_ext.TimeRange',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='start', full_name='gnmi_ext.TimeRange.start', index=0,
      number=1, type=3, cpp_type=2, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='end', full_name='gnmi_ext.TimeRange.end', index=1,
      number=2, type=3, cpp_type=2, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=625,
  serialized_end=664,
)


_COMMIT = _descriptor.Descriptor(
  name='Commit',
  full_name='gnmi_ext.Commit',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='id', full_name='gnmi_ext.Commit.id', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='commit', full_name='gnmi_ext.Commit.commit', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='confirm', full_name='gnmi_ext.Commit.confirm', index=2,
      number=3, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='cancel', full_name='gnmi_ext.Commit.cancel', index=3,
      number=4, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='set_rollback_duration', full_name='gnmi_ext.Commit.set_rollback_duration', index=4,
      number=5, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
    _descriptor.OneofDescriptor(
      name='action', full_name='gnmi_ext.Commit.action',
      index=0, containing_type=None, fields=[]),
  ],
  serialized_start=667,
  serialized_end=896,
)


_COMMITREQUEST = _descriptor.Descriptor(
  name='CommitRequest',
  full_name='gnmi_ext.CommitRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='rollback_duration', full_name='gnmi_ext.CommitRequest.rollback_duration', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=898,
  serialized_end=967,
)


_COMMITCONFIRM = _descriptor.Descriptor(
  name='CommitConfirm',
  full_name='gnmi_ext.CommitConfirm',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=969,
  serialized_end=984,
)


_COMMITCANCEL = _descriptor.Descriptor(
  name='CommitCancel',
  full_name='gnmi_ext.CommitCancel',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=986,
  serialized_end=1000,
)


_COMMITSETROLLBACKDURATION = _descriptor.Descriptor(
  name='CommitSetRollbackDuration',
  full_name='gnmi_ext.CommitSetRollbackDuration',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='rollback_duration', full_name='gnmi_ext.CommitSetRollbackDuration.rollback_duration', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1002,
  serialized_end=1083,
)


_DEPTH = _descriptor.Descriptor(
  name='Depth',
  full_name='gnmi_ext.Depth',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='level', full_name='gnmi_ext.Depth.level', index=0,
      number=1, type=13, cpp_type=3, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1085,
  serialized_end=1107,
)

_EXTENSION.fields_by_name['registered_ext'].message_type = _REGISTEREDEXTENSION
_EXTENSION.fields_by_name['master_arbitration'].message_type = _MASTERARBITRATION
_EXTENSION.fields_by_name['history'].message_type = _HISTORY
_EXTENSION.fields_by_name['commit'].message_type = _COMMIT
_EXTENSION.fields_by_name['depth'].message_type = _DEPTH
_EXTENSION.oneofs_by_name['ext'].fields.append(
  _EXTENSION.fields_by_name['registered_ext'])
_EXTENSION.fields_by_name['registered_ext'].containing_oneof = _EXTENSION.oneofs_by_name['ext']
_EXTENSION.oneofs_by_name['ext'].fields.append(
  _EXTENSION.fields_by_name['master_arbitration'])
_EXTENSION.fields_by_name['master_arbitration'].containing_oneof = _EXTENSION.oneofs_by_name['ext']
_EXTENSION.oneofs_by_name['ext'].fields.append(
  _EXTENSION.fields_by_name['history'])
_EXTENSION.fields_by_name['history'].containing_oneof = _EXTENSION.oneofs_by_name['ext']
_EXTENSION.oneofs_by_name['ext'].fields.append(
  _EXTENSION.fields_by_name['commit'])
_EXTENSION.fields_by_name['commit'].containing_oneof = _EXTENSION.oneofs_by_name['ext']
_EXTENSION.oneofs_by_name['ext'].fields.append(
  _EXTENSION.fields_by_name['depth'])
_EXTENSION.fields_by_name['depth'].containing_oneof = _EXTENSION.oneofs_by_name['ext']
_REGISTEREDEXTENSION.fields_by_name['id'].enum_type = _EXTENSIONID
_MASTERARBITRATION.fields_by_name['role'].message_type = _ROLE
_MASTERARBITRATION.fields_by_name['election_id'].message_type = _UINT128
_HISTORY.fields_by_name['range'].message_type = _TIMERANGE
_HISTORY.oneofs_by_name['request'].fields.append(
  _HISTORY.fields_by_name['snapshot_time'])
_HISTORY.fields_by_name['snapshot_time'].containing_oneof = _HISTORY.oneofs_by_name['request']
_HISTORY.oneofs_by_name['request'].fields.append(
  _HISTORY.fields_by_name['range'])
_HISTORY.fields_by_name['range'].containing_oneof = _HISTORY.oneofs_by_name['request']
_COMMIT.fields_by_name['commit'].message_type = _COMMITREQUEST
_COMMIT.fields_by_name['confirm'].message_type = _COMMITCONFIRM
_COMMIT.fields_by_name['cancel'].message_type = _COMMITCANCEL
_COMMIT.fields_by_name['set_rollback_duration'].message_type = _COMMITSETROLLBACKDURATION
_COMMIT.oneofs_by_name['action'].fields.append(
  _COMMIT.fields_by_name['commit'])
_COMMIT.fields_by_name['commit'].containing_oneof = _COMMIT.oneofs_by_name['action']
_COMMIT.oneofs_by_name['action'].fields.append(
  _COMMIT.fields_by_name['confirm'])
_COMMIT.fields_by_name['confirm'].containing_oneof = _COMMIT.oneofs_by_name['action']
_COMMIT.oneofs_by_name['action'].fields.append(
  _COMMIT.fields_by_name['cancel'])
_COMMIT.fields_by_name['cancel'].containing_oneof = _COMMIT.oneofs_by_name['action']
_COMMIT.oneofs_by_name['action'].fields.append(
  _COMMIT.fields_by_name['set_rollback_duration'])
_COMMIT.fields_by_name['set_rollback_duration'].containing_oneof = _COMMIT.oneofs_by_name['action']
_COMMITREQUEST.fields_by_name['rollback_duration'].message_type = google_dot_protobuf_dot_duration__pb2._DURATION
_COMMITSETROLLBACKDURATION.fields_by_name['rollback_duration'].message_type = google_dot_protobuf_dot_duration__pb2._DURATION
DESCRIPTOR.message_types_by_name['Extension'] = _EXTENSION
DESCRIPTOR.message_types_by_name['RegisteredExtension'] = _REGISTEREDEXTENSION
DESCRIPTOR.message_types_by_name['MasterArbitration'] = _MASTERARBITRATION
DESCRIPTOR.message_types_by_name['Uint128'] = _UINT128
DESCRIPTOR.message_types_by_name['Role'] = _ROLE
DESCRIPTOR.message_types_by_name['History'] = _HISTORY
DESCRIPTOR.message_types_by_name['TimeRange'] = _TIMERANGE
DESCRIPTOR.message_types_by_name['Commit'] = _COMMIT
DESCRIPTOR.message_types_by_name['CommitRequest'] = _COMMITREQUEST
DESCRIPTOR.message_types_by_name['CommitConfirm'] = _COMMITCONFIRM
DESCRIPTOR.message_types_by_name['CommitCancel'] = _COMMITCANCEL
DESCRIPTOR.message_types_by_name['CommitSetRollbackDuration'] = _COMMITSETROLLBACKDURATION
DESCRIPTOR.message_types_by_name['Depth'] = _DEPTH
DESCRIPTOR.enum_types_by_name['ExtensionID'] = _EXTENSIONID
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

Extension = _reflection.GeneratedProtocolMessageType('Extension', (_message.Message,), dict(
  DESCRIPTOR = _EXTENSION,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.Extension)
  ))
_sym_db.RegisterMessage(Extension)

RegisteredExtension = _reflection.GeneratedProtocolMessageType('RegisteredExtension', (_message.Message,), dict(
  DESCRIPTOR = _REGISTEREDEXTENSION,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.RegisteredExtension)
  ))
_sym_db.RegisterMessage(RegisteredExtension)

MasterArbitration = _reflection.GeneratedProtocolMessageType('MasterArbitration', (_message.Message,), dict(
  DESCRIPTOR = _MASTERARBITRATION,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.MasterArbitration)
  ))
_sym_db.RegisterMessage(MasterArbitration)

Uint128 = _reflection.GeneratedProtocolMessageType('Uint128', (_message.Message,), dict(
  DESCRIPTOR = _UINT128,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.Uint128)
  ))
_sym_db.RegisterMessage(Uint128)

Role = _reflection.GeneratedProtocolMessageType('Role', (_message.Message,), dict(
  DESCRIPTOR = _ROLE,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.Role)
  ))
_sym_db.RegisterMessage(Role)

History = _reflection.GeneratedProtocolMessageType('History', (_message.Message,), dict(
  DESCRIPTOR = _HISTORY,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.History)
  ))
_sym_db.RegisterMessage(History)

TimeRange = _reflection.GeneratedProtocolMessageType('TimeRange', (_message.Message,), dict(
  DESCRIPTOR = _TIMERANGE,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.TimeRange)
  ))
_sym_db.RegisterMessage(TimeRange)

Commit = _reflection.GeneratedProtocolMessageType('Commit', (_message.Message,), dict(
  DESCRIPTOR = _COMMIT,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.Commit)
  ))
_sym_db.RegisterMessage(Commit)

CommitRequest = _reflection.GeneratedProtocolMessageType('CommitRequest', (_message.Message,), dict(
  DESCRIPTOR = _COMMITREQUEST,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.CommitRequest)
  ))
_sym_db.RegisterMessage(CommitRequest)

CommitConfirm = _reflection.GeneratedProtocolMessageType('CommitConfirm', (_message.Message,), dict(
  DESCRIPTOR = _COMMITCONFIRM,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.CommitConfirm)
  ))
_sym_db.RegisterMessage(CommitConfirm)

CommitCancel = _reflection.GeneratedProtocolMessageType('CommitCancel', (_message.Message,), dict(
  DESCRIPTOR = _COMMITCANCEL,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.CommitCancel)
  ))
_sym_db.RegisterMessage(CommitCancel)

CommitSetRollbackDuration = _reflection.GeneratedProtocolMessageType('CommitSetRollbackDuration', (_message.Message,), dict(
  DESCRIPTOR = _COMMITSETROLLBACKDURATION,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.CommitSetRollbackDuration)
  ))
_sym_db.RegisterMessage(CommitSetRollbackDuration)

Depth = _reflection.GeneratedProtocolMessageType('Depth', (_message.Message,), dict(
  DESCRIPTOR = _DEPTH,
  __module__ = 'proto.gnmi_ext.gnmi_ext_pb2'
  # @@protoc_insertion_point(class_scope:gnmi_ext.Depth)
  ))
_sym_db.RegisterMessage(Depth)


DESCRIPTOR.has_options = True
DESCRIPTOR._options = _descriptor._ParseOptions(descriptor_pb2.FileOptions(), _b('Z)github.com/openconfig/gnmi/proto/gnmi_ext'))
# @@protoc_insertion_point(module_scope)
