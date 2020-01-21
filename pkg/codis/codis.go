// Tideland CoDis
//
// Copyright (C) 2019-2020 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license

package codis // import "tideland.dev/codis/pkg/codis"

//--------------------
// IMPORTS
//--------------------

import (
	"context"
	"fmt"
	"log"
	"time"

	codisv1alpha1 "tideland.dev/codis/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

//--------------------
// CONFIGURATION DISTRIBUTION
//--------------------

// ConfigurationDistributor implements the configuration distribution engine.
type ConfigurationDistributor struct {
	config        *rest.Config
	client        kubernetes.Interface
	namespace     string
	rulename      string
	ruleInterface codisv1alpha1.RuleInterface
	rule          *codisv1alpha1.ConfigurationDistributionRule
	ruleInformer  cache.SharedIndexInformer
	cmInformer    cache.SharedIndexInformer
	scrtInformer  cache.SharedIndexInformer
	nsInformer    cache.SharedIndexInformer
}

// New creates a new configuration distribution engine.
func New(config *rest.Config, namespace, rulename string) (*ConfigurationDistributor, error) {
	cd := &ConfigurationDistributor{
		config:    config,
		namespace: namespace,
		rulename:  rulename,
	}
	// Init rule interface.
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
	// Init client.
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot connect cluster: %v", err)
	}
	cd.client = client
	// Init informers.
	cd.ruleInformer = codisv1alpha1.NewRuleInformerWithInterface(cd.ruleInterface).Informer()
	factory := informers.NewSharedInformerFactory(cd.client, 30*time.Second)
	cd.cmInformer = factory.Core().V1().ConfigMaps().Informer()
	cd.scrtInformer = factory.Core().V1().Secrets().Informer()
	cd.nsInformer = factory.Core().V1().Namespaces().Informer()
	return cd, nil
}

// Run executes the configuration distributor.
func (cd *ConfigurationDistributor) Run(ctx context.Context) {
	cd.ruleInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cd.addRuleHandler,
		UpdateFunc: cd.updateRuleHandler,
		DeleteFunc: cd.deleteRuleHandler,
	})
	cd.cmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cd.addConfigMapHandler,
		UpdateFunc: cd.updateConfigMapHandler,
	})
	cd.scrtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cd.addSecretHandler,
		UpdateFunc: cd.updateSecretHandler,
	})
	cd.nsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: cd.addNamespaceHandler,
	})

	go cd.ruleInformer.Run(wait.NeverStop)
	go cd.cmInformer.Run(wait.NeverStop)
	go cd.scrtInformer.Run(wait.NeverStop)
	go cd.nsInformer.Run(wait.NeverStop)

	select {
	case <-ctx.Done():
		// Work is done.
	}
}

// addRuleHandler handles the adding of rules.
func (cd *ConfigurationDistributor) addRuleHandler(obj interface{}) {
	rule := obj.(*codisv1alpha1.ConfigurationDistributionRule)
	if rule.GetNamespace() != cd.namespace || rule.GetName() != cd.rulename {
		return
	}
	log.Printf("adding rule '%s' in namespace '%s' ...", rule.GetName(), rule.GetNamespace())
	cd.rule = rule
	cd.distributeAll()
}

// updateRuleHandler handles the updating of rules.
func (cd *ConfigurationDistributor) updateRuleHandler(oldobj, newobj interface{}) {
	oldrule := oldobj.(*codisv1alpha1.ConfigurationDistributionRule)
	newrule := newobj.(*codisv1alpha1.ConfigurationDistributionRule)
	if newrule.GetNamespace() != cd.namespace || newrule.GetName() != cd.rulename {
		return
	}
	if oldrule.GetResourceVersion() == newrule.GetResourceVersion() {
		return
	}
	log.Printf("updating rule '%s' in namespace '%s' ...", newrule.GetName(), newrule.GetNamespace())
	cd.rule = newrule
	cd.distributeAll()
}

// deleteRuleHandler handles the deleting of rules.
func (cd *ConfigurationDistributor) deleteRuleHandler(obj interface{}) {
	rule := obj.(*codisv1alpha1.ConfigurationDistributionRule)
	if rule.GetNamespace() != cd.namespace || rule.GetName() != cd.rulename {
		return
	}
	log.Printf("deleting rule '%s' in namespace '%s' ...", rule.GetName(), rule.GetNamespace())
	cd.rule = nil
}

// distributeAll copies all config maps and secrets to the namespaces of the rule.
func (cd *ConfigurationDistributor) distributeAll() {
	distributeAllOf := func(resource string) error {
		// TODO
		return nil
	}
	if err := distributeAllOf("configmap"); err != nil {
		log.Printf("cannot copy all configmaps: %v", err)
	}
	if err := distributeAllOf("secret"); err != nil {
		log.Printf("cannot copy all configmaps: %v", err)
	}
}

// addConfigMapHandler handles the adding of ConfigMaps.
func (cd *ConfigurationDistributor) addConfigMapHandler(obj interface{}) {
	if cd.rule == nil {
		return
	}
	if cd.rule.Spec.Mode != "configmap" && cd.rule.Spec.Mode != "both" {
		return
	}
	cm := obj.(*corev1.ConfigMap)
	if cm.GetNamespace() != cd.namespace {
		return
	}
	if cd.rule.Spec.Selector != "" {
		if cm.GetLabels()["rule"] != cd.rule.Spec.Selector {
			return
		}
	}
	cd.applyConfigMap(cm, true)
}

// updateConfigMapHandler handles the updating of ConfigMaps.
func (cd *ConfigurationDistributor) updateConfigMapHandler(oldobj, newobj interface{}) {
	if cd.rule == nil {
		return
	}
	if cd.rule.Spec.Mode != "configmap" && cd.rule.Spec.Mode != "both" {
		return
	}
	oldcm := oldobj.(*corev1.ConfigMap)
	newcm := newobj.(*corev1.ConfigMap)
	if newcm.GetNamespace() != cd.namespace || oldcm.GetResourceVersion() == newcm.GetResourceVersion() {
		return
	}
	if cd.rule.Spec.Selector != "" {
		if newcm.GetLabels()["rule"] != cd.rule.Spec.Selector {
			return
		}
	}
	cd.applyConfigMap(newcm, false)
}

// applyConfigMap applies the ConfigMap to the namespaces configured in the distributor.
func (cd *ConfigurationDistributor) applyConfigMap(in *corev1.ConfigMap, create bool) {
	log.Printf("applying 'configmap/%s' ...", in.GetName())
	for _, namespace := range cd.rule.Spec.Namespaces {
		cmInf := cd.client.CoreV1().ConfigMaps(namespace)
		out := in.DeepCopy()
		out.SetNamespace(namespace)
		out.SetResourceVersion("")
		out.SetUID("")

		var err error
		if create {
			_, err = cmInf.Create(out)
		} else {
			_, err = cmInf.Update(out)
		}
		if err != nil {
			log.Printf(
				"cannot apply 'configmap/%s' to namespace '%s': %v",
				in.GetName(),
				namespace,
				err,
			)
		}
	}
}

// addSecretHandler handles the adding of Secrets.
func (cd *ConfigurationDistributor) addSecretHandler(obj interface{}) {
	if cd.rule == nil {
		return
	}
	if cd.rule.Spec.Mode != "secret" && cd.rule.Spec.Mode != "both" {
		return
	}
	scrt := obj.(*corev1.Secret)
	if scrt.GetNamespace() != cd.namespace {
		return
	}
	if cd.rule.Spec.Selector != "" {
		if scrt.GetLabels()["rule"] != cd.rule.Spec.Selector {
			return
		}
	}
	cd.applySecret(scrt, true)
}

// updateSecretHandler handles the updating of Secrets.
func (cd *ConfigurationDistributor) updateSecretHandler(oldobj, newobj interface{}) {
	if cd.rule == nil {
		return
	}
	if cd.rule.Spec.Mode != "secret" && cd.rule.Spec.Mode != "both" {
		return
	}
	oldscrt := oldobj.(*corev1.Secret)
	newscrt := newobj.(*corev1.Secret)
	if newscrt.GetNamespace() != cd.namespace || oldscrt.GetResourceVersion() == newscrt.GetResourceVersion() {
		return
	}
	if cd.rule.Spec.Selector != "" {
		if newscrt.GetLabels()["rule"] != cd.rule.Spec.Selector {
			return
		}
	}
	cd.applySecret(newscrt, false)
}

// applySecret applies the Secret to the namespaces configured in the distributor.
func (cd *ConfigurationDistributor) applySecret(in *corev1.Secret, create bool) {
	log.Printf("applying 'secret/%s' ...", in.GetName())
	for _, namespace := range cd.rule.Spec.Namespaces {
		scrtInf := cd.client.CoreV1().Secrets(namespace)
		out := in.DeepCopy()
		out.SetNamespace(namespace)
		out.SetResourceVersion("")
		out.SetUID("")

		var err error
		if create {
			_, err = scrtInf.Create(out)
		} else {
			_, err = scrtInf.Update(out)
		}
		if err != nil {
			log.Printf(
				"cannot apply 'secret/%s' to namespace '%s': %v",
				in.GetName(),
				namespace,
				err,
			)
		}
	}
}

// addNamespaceHandler handles the adding of Namespaces.
func (cd *ConfigurationDistributor) addNamespaceHandler(obj interface{}) {
	if cd.rule == nil {
		return
	}
	ns := obj.(*corev1.Namespace)
	for _, namespace := range cd.rule.Spec.Namespaces {
		if ns.GetName() == namespace {
			// Namespace in rule.
			cd.applyMatchingConfigMaps(ns.GetName())
			cd.applyMatchingSecrets(ns.GetName())
			return
		}
	}
}

// applyMatchingConfigMaps applies the matching ConfigMaps in own Namespace to
// the given Namespace.
func (cd *ConfigurationDistributor) applyMatchingConfigMaps(namespace string) {
}

// applyMatchingSecrets applies the matching Secrets in own Namespace to
// the given Namespace.
func (cd *ConfigurationDistributor) applyMatchingSecrets(namespace string) {
}

// EOF
