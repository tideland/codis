// Tideland CoDis
//
// Copyright (C) 2019-2020 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license

package v1alpha1 // import "tideland.dev/codis/api/v1alpha1"

//--------------------
// IMPORTS
//--------------------

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

//--------------------
// RULE LISTER
//--------------------

// RuleLister helps list Rules.
type RuleLister interface {
	List(selector labels.Selector) ([]*ConfigurationDistributionRule, error)
	Get(name string) (*ConfigurationDistributionRule, error)
}

// ruleLister implements RuleLister.
type ruleLister struct {
	rif RuleInterface
}

// List implements RuleLister.
func (rl *ruleLister) List(selector labels.Selector) ([]*ConfigurationDistributionRule, error) {
	cdrl, err := rl.rif.List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	list := make([]*ConfigurationDistributionRule, len(cdrl.Items))
	for i, rule := range cdrl.Items {
		list[i] = &rule
	}
	return list, nil
}

// Get implements RuleLister.
func (rl *ruleLister) Get(name string) (*ConfigurationDistributionRule, error) {
	return rl.rif.Get(name, metav1.GetOptions{})
}

//--------------------
// RULE INFORMER
//--------------------

// RuleInformer provides access to a shared informer and lister for Rules.
type RuleInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() RuleLister
}

// ruleInformer implements RuleInformer.
type ruleInformer struct {
	rif RuleInterface
}

// NewRuleInformerWithInterface creates a new rule informer instance
// based on a given interface.
func NewRuleInformerWithInterface(rif RuleInterface) RuleInformer {
	return &ruleInformer{
		rif: rif,
	}
}

// Informer implements RuleInformer.
func (ri *ruleInformer) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (result runtime.Object, err error) {
				return ri.rif.List(opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return ri.rif.Watch(opts)
			},
		},
		&ConfigurationDistributionRule{},
		30*time.Second,
		cache.Indexers{},
	)
}

// Lister implements RuleInformer.
func (ri *ruleInformer) Lister() RuleLister {
	return &ruleLister{
		rif: ri.rif,
	}
}

// EOF
