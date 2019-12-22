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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

//--------------------
// INTERFACES
//--------------------

// RuleInterface defines the interface to access a rule.
type RuleInterface interface {
	List(opts metav1.ListOptions) (*ConfigurationDistributionRuleList, error)
	Get(name string, opts metav1.GetOptions) (*ConfigurationDistributionRule, error)
	Create(copier *ConfigurationDistributionRule, opts metav1.CreateOptions) (*ConfigurationDistributionRule, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

// NamespaceableRuleInterface defines the interface to access a rule in a namespace.
type NamespaceableRuleInterface interface {
	Namespace(namespace string) RuleInterface
}

// namespaceableRuleInterface implements NamespaceableRuleInterface.
type namespaceableRuleInterface struct {
	restClient rest.Interface
}

// NewForConfig creates a client for accessing the rules in a namespace.
func NewForConfig(config *rest.Config) (NamespaceableRuleInterface, error) {
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: groupName, Version: groupVersion}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	restClient, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		return nil, err
	}

	return &namespaceableRuleInterface{
		restClient: restClient,
	}, nil
}

// Namespace implements the NamespaceableRuleInterface.
func (nri *namespaceableRuleInterface) Namespace(namespace string) RuleInterface {
	return &ruleInterface{
		restClient: nri.restClient,
		namespace:  namespace,
	}
}

//--------------------
// RULE INTERFACE IMPLEMENTATION
//--------------------

// ruleInterface implements ConfigurationDistributionRuleInterface.
type ruleInterface struct {
	restClient rest.Interface
	namespace  string
}

// List implements ConfigurationDistributionRuleClient.
func (ri *ruleInterface) List(opts metav1.ListOptions) (*ConfigurationDistributionRuleList, error) {
	result := ConfigurationDistributionRuleList{}
	err := ri.restClient.
		Get().
		Namespace(ri.namespace).
		Resource("configurationdistributionrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

// Get implements ConfigurationDistributionRuleClient.
func (ri *ruleInterface) Get(name string, opts metav1.GetOptions) (*ConfigurationDistributionRule, error) {
	result := ConfigurationDistributionRule{}
	err := ri.restClient.
		Get().
		Namespace(ri.namespace).
		Resource("configurationdistributionrules").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

// Create implements ConfigurationDistributionRuleClient.
func (ri *ruleInterface) Create(rule *ConfigurationDistributionRule, opts metav1.CreateOptions) (*ConfigurationDistributionRule, error) {
	result := ConfigurationDistributionRule{}
	err := ri.restClient.
		Post().
		Namespace(ri.namespace).
		Resource("configurationdistributionrules").
		Body(rule).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

// Watch implements ConfigurationDistributionRuleClient.
func (ri *ruleInterface) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return ri.restClient.
		Get().
		Namespace(ri.namespace).
		Resource("configurationdistributionrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// EOF
