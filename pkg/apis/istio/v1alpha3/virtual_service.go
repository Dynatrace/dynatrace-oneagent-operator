package v1alpha3

import (
	"bufio"
	"bytes"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"

	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualService is an Istio VirtualService resource
type VirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualServiceSpec `json:"spec"`
}

func (vs *VirtualService) GetSpecMessage() proto.Message {
	return &vs.Spec.VirtualService
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualServiceList is a list of VirtualService resources
type VirtualServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualService `json:"items"`
}

// VirtualServiceSpec is a wrapper around Istio VirtualService
type VirtualServiceSpec struct {
	istiov1alpha3.VirtualService
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
// Based of https://github.com/istio/istio/blob/release-0.8/pilot/pkg/config/kube/crd/types.go#L450
func (in *VirtualServiceSpec) DeepCopyInto(out *VirtualServiceSpec) {
	*out = *in
}

func (vs *VirtualServiceSpec) MarshalJSON() ([]byte, error) {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	marshaler := jsonpb.Marshaler{}
	err := marshaler.Marshal(writer, &vs.VirtualService)
	if err != nil {
		return nil, err
	}

	writer.Flush()
	return buffer.Bytes(), nil
}

func (vs *VirtualServiceSpec) UnmarshalJSON(b []byte) error {
	reader := bytes.NewReader(b)
	unmarshaler := jsonpb.Unmarshaler{}
	err := unmarshaler.Unmarshal(reader, &vs.VirtualService)
	if err != nil {
		return err
	}
	return nil
}
