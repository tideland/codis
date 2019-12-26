// Tideland CoDis
//
// Copyright (C) 2019 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license

package main // import "tideland.dev/codis/cmd/codis"

//--------------------
// IMPORTS
//--------------------

import (
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	"tideland.dev/codis/pkg/codis"
)

//--------------------
// MAIN
//--------------------

func main() {
	log.Printf("Starting the Tideland Configuration Distributor for Kubernetes (CoDis) ...")

	// Namespace and rulename.
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	rulename := os.Getenv("RULENAME")
	if rulename == "" {
		rulename = "default-rule"
	}

	// Configuration.
	cfgFilename := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", cfgFilename)
	if err != nil {
		log.Fatalf("Cannot read controller configuration: %v", err)
	}

	// Configuration distributor.
	cd, err := codis.New(config, namespace, rulename)
	if err != nil {
		log.Fatalf("Cannot init configuration distributor: %v", err)
	}
	log.Printf("Run the configuration distributor ...")
	err = cd.Do()
	if err != nil {
		log.Fatalf("Error during running the configuration distributor: %v", err)
	}

	log.Printf("Done!")
}

// EOF
