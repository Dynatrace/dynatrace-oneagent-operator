// +build !ignore_autogenerated

/*
Copyright 2020 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by operator-sdk-v0.17.1-x86_64-apple-darwin. DO NOT EDIT.

package v1alpha1

import (
	status "github.com/operator-framework/operator-sdk/pkg/status"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BaseOneAgentSpec) DeepCopyInto(out *BaseOneAgentSpec) {
	*out = *in
	if in.Proxy != nil {
		in, out := &in.Proxy, &out.Proxy
		*out = new(OneAgentProxy)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BaseOneAgentSpec.
func (in *BaseOneAgentSpec) DeepCopy() *BaseOneAgentSpec {
	if in == nil {
		return nil
	}
	out := new(BaseOneAgentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BaseOneAgentStatus) DeepCopyInto(out *BaseOneAgentStatus) {
	*out = *in
	in.UpdatedTimestamp.DeepCopyInto(&out.UpdatedTimestamp)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(status.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.LastAPITokenProbeTimestamp != nil {
		in, out := &in.LastAPITokenProbeTimestamp, &out.LastAPITokenProbeTimestamp
		*out = (*in).DeepCopy()
	}
	if in.LastPaaSTokenProbeTimestamp != nil {
		in, out := &in.LastPaaSTokenProbeTimestamp, &out.LastPaaSTokenProbeTimestamp
		*out = (*in).DeepCopy()
	}
	in.LastClusterVersionProbeTimestamp.DeepCopyInto(&out.LastClusterVersionProbeTimestamp)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BaseOneAgentStatus.
func (in *BaseOneAgentStatus) DeepCopy() *BaseOneAgentStatus {
	if in == nil {
		return nil
	}
	out := new(BaseOneAgentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgent) DeepCopyInto(out *OneAgent) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgent.
func (in *OneAgent) DeepCopy() *OneAgent {
	if in == nil {
		return nil
	}
	out := new(OneAgent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OneAgent) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentAPM) DeepCopyInto(out *OneAgentAPM) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentAPM.
func (in *OneAgentAPM) DeepCopy() *OneAgentAPM {
	if in == nil {
		return nil
	}
	out := new(OneAgentAPM)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OneAgentAPM) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentAPMList) DeepCopyInto(out *OneAgentAPMList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OneAgentAPM, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentAPMList.
func (in *OneAgentAPMList) DeepCopy() *OneAgentAPMList {
	if in == nil {
		return nil
	}
	out := new(OneAgentAPMList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OneAgentAPMList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentAPMSpec) DeepCopyInto(out *OneAgentAPMSpec) {
	*out = *in
	in.BaseOneAgentSpec.DeepCopyInto(&out.BaseOneAgentSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentAPMSpec.
func (in *OneAgentAPMSpec) DeepCopy() *OneAgentAPMSpec {
	if in == nil {
		return nil
	}
	out := new(OneAgentAPMSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentAPMStatus) DeepCopyInto(out *OneAgentAPMStatus) {
	*out = *in
	in.BaseOneAgentStatus.DeepCopyInto(&out.BaseOneAgentStatus)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentAPMStatus.
func (in *OneAgentAPMStatus) DeepCopy() *OneAgentAPMStatus {
	if in == nil {
		return nil
	}
	out := new(OneAgentAPMStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentInstance) DeepCopyInto(out *OneAgentInstance) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentInstance.
func (in *OneAgentInstance) DeepCopy() *OneAgentInstance {
	if in == nil {
		return nil
	}
	out := new(OneAgentInstance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentList) DeepCopyInto(out *OneAgentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OneAgent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentList.
func (in *OneAgentList) DeepCopy() *OneAgentList {
	if in == nil {
		return nil
	}
	out := new(OneAgentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OneAgentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentProxy) DeepCopyInto(out *OneAgentProxy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentProxy.
func (in *OneAgentProxy) DeepCopy() *OneAgentProxy {
	if in == nil {
		return nil
	}
	out := new(OneAgentProxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentSpec) DeepCopyInto(out *OneAgentSpec) {
	*out = *in
	in.BaseOneAgentSpec.DeepCopyInto(&out.BaseOneAgentSpec)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.WaitReadySeconds != nil {
		in, out := &in.WaitReadySeconds, &out.WaitReadySeconds
		*out = new(uint16)
		**out = **in
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentSpec.
func (in *OneAgentSpec) DeepCopy() *OneAgentSpec {
	if in == nil {
		return nil
	}
	out := new(OneAgentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentStatus) DeepCopyInto(out *OneAgentStatus) {
	*out = *in
	in.BaseOneAgentStatus.DeepCopyInto(&out.BaseOneAgentStatus)
	if in.Instances != nil {
		in, out := &in.Instances, &out.Instances
		*out = make(map[string]OneAgentInstance, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentStatus.
func (in *OneAgentStatus) DeepCopy() *OneAgentStatus {
	if in == nil {
		return nil
	}
	out := new(OneAgentStatus)
	in.DeepCopyInto(out)
	return out
}
