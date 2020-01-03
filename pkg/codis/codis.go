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
	"context"
	"fmt"
	"log"
	"time"

	codisv1alpha1 "tideland.dev/codis/pkg/v1alpha1"

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
	factory := informers.NewSharedInformerFactory(cd.client, time.Second*30)
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

	go cd.ruleInformer.Run(wait.NeverStop)
	go cd.cmInformer.Run(wait.NeverStop)
	go cd.scrtInformer.Run(wait.NeverStop)

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
	cd.rule = rule
	cd.copyAll()
}

// updateRuleHandler handles the updating of rules.
func (cd *ConfigurationDistributor) updateRuleHandler(oldobj, newobj interface{}) {
	rule := newobj.(*codisv1alpha1.ConfigurationDistributionRule)
	if rule.GetNamespace() != cd.namespace || rule.GetName() != cd.rulename {
		return
	}
	cd.rule = rule
	cd.copyAll()
}

// deleteRuleHandler handles the deleting of rules.
func (cd *ConfigurationDistributor) deleteRuleHandler(obj interface{}) {
	rule := obj.(*codisv1alpha1.ConfigurationDistributionRule)
	if rule.GetNamespace() != cd.namespace || rule.GetName() != cd.rulename {
		return
	}
	cd.rule = nil
}

// addConfigMapHandler handles the adding of ConfigMaps.
func (cd *ConfigurationDistributor) addConfigMapHandler(obj interface{}) {
	if cd.rule == nil {
		return
	}
	if cd.rule.Spec.Kind != "configmap" && cd.rule.Spec.Kind != "both" {
		return
	}
	cm := obj.(*corev1.ConfigMap)
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
	if cd.rule.Spec.Kind != "configmap" && cd.rule.Spec.Kind != "both" {
		return
	}
	cm := newobj.(*corev1.ConfigMap)
	if cd.rule.Spec.Selector != "" {
		if cm.GetLabels()["rule"] != cd.rule.Spec.Selector {
			return
		}
	}
	cd.applyConfigMap(cm, false)
}

// applyConfigMap applies the ConfigMap to the namespaces configured in the distributor.
func (cd *ConfigurationDistributor) applyConfigMap(in *corev1.ConfigMap, create bool) {
	log.Printf("applying 'configmap/%s' ...", in.GetName())
	for _, namespace := range cd.rule.Spec.Namespaces {
		cmInf := cd.client.CoreV1().ConfigMaps(namespace)
		out := in.DeepCopy()
		out.SetNamespace(namespace)

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
	if cd.rule.Spec.Kind != "secret" && cd.rule.Spec.Kind != "both" {
		return
	}
	scrt := obj.(*corev1.Secret)
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
	if cd.rule.Spec.Kind != "secret" && cd.rule.Spec.Kind != "both" {
		return
	}
	scrt := newobj.(*corev1.Secret)
	if cd.rule.Spec.Selector != "" {
		if scrt.GetLabels()["rule"] != cd.rule.Spec.Selector {
			return
		}
	}
	cd.applySecret(scrt, false)
}

// applySecret applies the Secret to the namespaces configured in the distributor.
func (cd *ConfigurationDistributor) applySecret(in *corev1.Secret, create bool) {
	log.Printf("applying 'secret/%s' ...", in.GetName())
	for _, namespace := range cd.rule.Spec.Namespaces {
		scrtInf := cd.client.CoreV1().Secrets(namespace)
		out := in.DeepCopy()
		out.SetNamespace(namespace)

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

// copyAll copies all config maps and secrets to the namespaces of the rule.
func (cd *ConfigurationDistributor) copyAll() {
	copyAllOf := func(resource string) error {
		return nil
	}
	if err := copyAllOf("configmap"); err != nil {
		log.Printf("cannot copy all configmaps: %v", err)
	}
	if err := copyAllOf("secret"); err != nil {
		log.Printf("cannot copy all configmaps: %v", err)
	}
}

// EOF
