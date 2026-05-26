package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyObject implements runtime.Object for TacitoAgent.
func (in *TacitoAgent) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(TacitoAgent)
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.TypeMeta = out.TypeMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return out
}

// DeepCopyObject implements runtime.Object for TacitoAgentList.
func (in *TacitoAgentList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(TacitoAgentList)
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	out.TypeMeta = in.TypeMeta
	if in.Items != nil {
		out.Items = make([]TacitoAgent, len(in.Items))
		copy(out.Items, in.Items)
	}
	return out
}

// Ensure compile-time interface compliance.
var (
	_ runtime.Object = &TacitoAgent{}
	_ runtime.Object = &TacitoAgentList{}
)
