// +build !ignore_autogenerated

// Code generated by operator-sdk. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CoralogixLogger) DeepCopyInto(out *CoralogixLogger) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CoralogixLogger.
func (in *CoralogixLogger) DeepCopy() *CoralogixLogger {
	if in == nil {
		return nil
	}
	out := new(CoralogixLogger)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CoralogixLogger) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CoralogixLoggerList) DeepCopyInto(out *CoralogixLoggerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CoralogixLogger, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CoralogixLoggerList.
func (in *CoralogixLoggerList) DeepCopy() *CoralogixLoggerList {
	if in == nil {
		return nil
	}
	out := new(CoralogixLoggerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CoralogixLoggerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CoralogixLoggerSpec) DeepCopyInto(out *CoralogixLoggerSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CoralogixLoggerSpec.
func (in *CoralogixLoggerSpec) DeepCopy() *CoralogixLoggerSpec {
	if in == nil {
		return nil
	}
	out := new(CoralogixLoggerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CoralogixLoggerStatus) DeepCopyInto(out *CoralogixLoggerStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CoralogixLoggerStatus.
func (in *CoralogixLoggerStatus) DeepCopy() *CoralogixLoggerStatus {
	if in == nil {
		return nil
	}
	out := new(CoralogixLoggerStatus)
	in.DeepCopyInto(out)
	return out
}
