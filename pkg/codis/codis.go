// Tideland CoDis
//
// Copyright (C) 2019 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license

package codis // import "tideland.dev/codis/pkg/codis"

//--------------------
// IMPORTS
//--------------------

import (
	"fmt"
	"os"
	"os/signal"

	codisv1alpha1 "tideland.dev/codis/pkg/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

//--------------------
// CONFIGURATION DISTRIBUTION
//--------------------

// ConfigurationDistributor implements the configuration distribution engine.
type ConfigurationDistributor struct {
	config        *rest.Config
	namespace     string
	rulename      string
	ruleInterface codisv1alpha1.RuleInterface
	dynamicClient dynamic.Interface
	rule          *codisv1alpha1.ConfigurationDistributionRule
}

// New creates a new configuration distribution engine.
func New(config *rest.Config, namespace, rulename string) (*ConfigurationDistributor, error) {
	cd := &ConfigurationDistributor{
		config:    config,
		namespace: namespace,
		rulename:  rulename,
	}
	namespaceableRuleInterface, err := codisv1alpha1.NewForConfig(cd.config)
	if err != nil {
		return nil, fmt.Errorf("cannot create namespaceable rule interface: %v", err)
	}
	cd.ruleInterface = namespaceableRuleInterface.Namespace(namespace)
	rule, err := cd.ruleInterface.Get(cd.rulename, metav1.GetOptions{})
	if err != nil {
		// In case of an error the controller allows a later loading based
		// on an event.
		cd.rule = rule
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot connect cluster: %v", err)
	}
	cd.dynamicClient = dynamicClient
	return cd, nil
}

// Do executes the configuration distributor.
func (cd *ConfigurationDistributor) Do() error {
	// Create watches.
	cdrWatch, err := cd.createWatch("k8s.tideland.dev", "v1alpha1", "configurationdistributionrules")
	if err != nil {
		return fmt.Errorf("cannot create ConfigurationDistributionRule watch: %v", err)
	}
	cmWatch, err := cd.createWatch("core", "v1", "configmaps")
	if err != nil {
		return fmt.Errorf("cannot create ConfigMap watch: %v", err)
	}
	secretWatch, err := cd.createWatch("core", "v1", "secrets")
	if err != nil {
		return fmt.Errorf("cannot create Secret watch: %v", err)
	}
	nsWatch, err := cd.createWatch("core", "v1", "namespaces")
	if err != nil {
		return fmt.Errorf("cannot create Namespaces watch: %v", err)
	}
	// Also listen to interrupt signal.
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)
	// Wait for events.
	for {
		select {
		case <-stopChan:
			return nil
		case evt := <-cdrWatch.ResultChan():
			err = cd.handleConfigurationDistributionRule(evt)
		case evt := <-cmWatch.ResultChan():
			err = cd.handleConfigMap(evt)
		case evt := <-secretWatch.ResultChan():
			err = cd.handleSecret(evt)
		case evt := <-nsWatch.ResultChan():
			err = cd.handleNamespace(evt)
		}
		if err != nil {
			return fmt.Errorf("cannot handle event: %v", err)
		}
	}
}

// createWatch simplifies creates a watch based on group, version, and resource.
func (cd *ConfigurationDistributor) createWatch(group, version, resource string) (watch.Interface, error) {
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	nri := cd.dynamicClient.Resource(gvr)
	return nri.Watch(metav1.ListOptions{})
}

// handleConfigurationDistributionRule cares for events regarding the CDRs. We only care for the
// types ADDED, MODIFIED, and DELETED.
func (cd *ConfigurationDistributor) handleConfigurationDistributionRule(evt watch.Event) error {
	if evt.Type != watch.Added && evt.Type != watch.Modified && evt.Type != watch.Deleted {
		return nil
	}
	unstructuredObject := evt.Object.(*unstructured.Unstructured)
	if unstructuredObject.GetNamespace() != cd.namespace || unstructuredObject.GetName() != cd.rulename {
		return nil
	}
	if evt.Type == watch.Deleted {
		// Simple case of deleting the rule.
		cd.rule = nil
		return nil
	}
	// Add or update rule.
	rule, err := cd.ruleInterface.Get(cd.rulename, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot get configuration distribution rule '%s': %v", cd.rulename, err)
	}
	cd.rule = rule
	return cd.copyAll()
}

// handleConfigMap cares for events regarding config maps. We only care for the
// types ADDED and MODIFIED. Here config maps are copied to all configured
// namespaces.
func (cd *ConfigurationDistributor) handleConfigMap(evt watch.Event) {
	if cd.rule == nil {
		return nil
	}
	if evt.Type != watch.Added && evt.Type != watch.Modified {
		return nil
	}
	if !cd.matches(evt.Object, "configmap") {
		return nil
	}
	return cd.copy(evt.Object, "core", "v1", "configmaps")
}

// handleSecret cares for events regarding secrets. We only care for the
// types ADDED and MODIFIED. Here secrets are copied to all configured
// namespaces.
func (cd *ConfigurationDistributor) handleSecret(evt watch.Event) error {
	if cd.rule == nil {
		return nil
	}
	if evt.Type != watch.Added && evt.Type != watch.Modified {
		return nil
	}
	if !cd.matches(evt.Object, "secret") {
		return nil
	}
	return cd.copy(evt.Object, "core", "v1", "secrets")
}

// handleNamespace cares for events regarding namespaces. We only care for ADDED.
// Here the config maps and secrets are copied to all configured namespaces.
func (cd *ConfigurationDistributor) handleNamespace(evt watch.Event) error {
	if cd.rule == nil {
		return nil
	}
	if evt.Type != watch.Added {
		return nil
	}
	unstructuredObject := object.(*unstructured.Unstructured)
	isNamespace := false
	for _, namespace := range cd.rule.Spec.Namespaces {
		if unstructuredObject.GetNamespace() == namespace {
			isNamespace = true
		}
	}
	if isNamespace {
		return cd.copyAll()
	}
	return nil
}

// matches checks if the event matches to our copier.
func (cd *ConfigurationDistributor) matches(object runtime.Object, kind string) bool {
	// Kind.
	unstructuredObject := object.(*unstructured.Unstructured)
	if cd.rule.Spec.Kind != kind && cd.rule.Spec.Kind != "both" {
		// Kind is wrong.
		return false
	}
	// Namespace.
	if cd.rule.ObjectMeta.Namespace != unstructuredObject.GetNamespace() {
		// Namespace does not match.
		return false
	}
	// Selector.
	if cd.rule.Spec.Selector != "" {
		if unstructuredObject.GetLabels()["rule"] != cd.rule.Spec.Selector {
			// Rule-selector doesn't match to labels.
			return false
		}
	}
	return true
}

// copy copies the objects to the namespace configured in the copier.
func (cd *ConfigurationDistributor) copy(object runtime.Object, group, version, resource string) error {
	in := object.(*unstructured.Unstructured)
	client := cd.dynamicClient.Resource(schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	})
	for _, namespace := range cd.rule.Spec.Namespaces {
		out := in.DeepCopy()
		out.SetNamespace(namespace)

		_, err := client.Create(out, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf(
				"cannot create '%s/%s' in namespace '%s': %v",
				in.GetKind(),
				in.GetName(),
				namespace,
				err,
			)
		}
	}
	return nil
}

// copyAll copies all config maps and secrets to the namespaces of the rule.
func (cd *ConfigurationDistributor) copyAll() error {
	// TODO Implement copy.
	return nil
}
