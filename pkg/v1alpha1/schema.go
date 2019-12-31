// Tideland CoDis
//
// Copyright (C) 2019 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license

package v1alpha1 // import "tideland.dev/codis/pkg/v1alpha1"

//--------------------
// IMPORTS
//--------------------

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//--------------------
// CONSTANTS
//--------------------

const (
	groupName    = "k8s.tideland.dev"
	groupVersion = "v1alpha1"
)

var (
	// SchemeGroupVersion describes the CRD.
	SchemeGroupVersion = schema.GroupVersion{Group: groupName, Version: groupVersion}

	// SchemeBuilder creates a scheme builder for the known types.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme points to a function to create the known types.
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes adds our new types.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ConfigurationDistributionRule{},
		&ConfigurationDistributionRuleList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

//--------------------
// SCHEMA
//--------------------

// ConfigurationDistributionRuleSpec specifies one configuration distribution rule.
type ConfigurationDistributionRuleSpec struct {
	Kind       string   `json:"kind"`
	Selector   string   `json:"selector"`
	Namespaces []string `json:"namespaces"`
}

// ConfigurationDistributionRule contains the Kubernetes base informations and the spec.
type ConfigurationDistributionRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ConfigurationDistributionRuleSpec `json:"spec"`
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *ConfigurationDistributionRule) DeepCopyInto(out *ConfigurationDistributionRule) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = ConfigurationDistributionRuleSpec{
		Kind:       in.Spec.Kind,
		Selector:   in.Spec.Selector,
		Namespaces: make([]string, len(in.Spec.Namespaces)),
	}
	for i := range in.Spec.Namespaces {
		out.Spec.Namespaces[i] = in.Spec.Namespaces[i]
	}
}

// DeepCopyObject returns a generically typed copy of a rule.
func (in *ConfigurationDistributionRule) DeepCopyObject() runtime.Object {
	out := ConfigurationDistributionRule{}
	in.DeepCopyInto(&out)

	return &out
}

// ConfigurationDistributionRuleList contains the Kubernetes base informations and a list of copiers.
type ConfigurationDistributionRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ConfigurationDistributionRule `json:"items"`
}

// DeepCopyObject returns a generically typed copy of a rule list.
func (in *ConfigurationDistributionRuleList) DeepCopyObject() runtime.Object {
	out := ConfigurationDistributionRuleList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]ConfigurationDistributionRule, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// EOF
