// Tideland CoDis
//
// Copyright (C) 2019-2020 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license

package main // import "tideland.dev/codis/cmd/codis"

//--------------------
// IMPORTS
//--------------------

import (
	"context"
	"flag"
	"log"

	"tideland.dev/codis/pkg/codis"
	codisv1alpha1 "tideland.dev/codis/pkg/v1alpha1"

	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

//--------------------
// MAIN
//--------------------

func main() {
	log.Printf("Starting the Tideland Configuration Distributor (CoDis) ...")

	// Configuration.
	var (
		kubeconfig string
		masterURL  string
		namespace  string
		rulename   string
	)
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "Address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&namespace, "namespace", "default", "Namespace of the managed configuration distributor rule.")
	flag.StringVar(&rulename, "rulename", "default-rule", "Name of the managed configuration distributor rule.")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Cannot read controller configuration: %v", err)
	}

	// Configuration distributor.
	cd, err := codis.New(config, namespace, rulename)
	if err != nil {
		log.Fatalf("Cannot init configuration distributor: %v", err)
	}
	codisv1alpha1.AddToScheme(scheme.Scheme)

	log.Printf("Run the configuration distributor ...")
	cd.Run(context.Background())
}

// EOF
