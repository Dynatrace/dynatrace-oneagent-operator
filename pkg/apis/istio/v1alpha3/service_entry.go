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

// ServiceEntry is an Istio ServiceEntry resource
type ServiceEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceEntrySpec `json:"spec"`
}

func (vs *ServiceEntry) GetSpecMessage() proto.Message {
	return &vs.Spec.ServiceEntry
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceEntryList is a list of ServiceEntry resources
type ServiceEntryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ServiceEntry `json:"items"`
}

// ServiceEntrySpec is a wrapper around Istio ServiceEntry
type ServiceEntrySpec struct {
	istiov1alpha3.ServiceEntry
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
// Based of https://github.com/istio/istio/blob/release-0.8/pilot/pkg/config/kube/crd/types.go#L450
func (in *ServiceEntrySpec) DeepCopyInto(out *ServiceEntrySpec) {
	*out = *in
}

func (vs *ServiceEntrySpec) MarshalJSON() ([]byte, error) {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	marshaler := jsonpb.Marshaler{}
	err := marshaler.Marshal(writer, &vs.ServiceEntry)
	if err != nil {
		return nil, err
	}

	writer.Flush()
	return buffer.Bytes(), nil
}

func (vs *ServiceEntrySpec) UnmarshalJSON(b []byte) error {
	reader := bytes.NewReader(b)
	unmarshaler := jsonpb.Unmarshaler{}
	err := unmarshaler.Unmarshal(reader, &vs.ServiceEntry)
	if err != nil {
		return err
	}
	return nil
}
