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
)

//--------------------
// MAIN
//--------------------

func main() {
	log.Infof("Starting the Tideland Configuration Distributor for Kubernetes (CoDis)")

	// Namespace and rulename.
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	ruleName := os.Getenv("RULENAME")
	if ruleName == "" {
		ruleName = "default-rule"
	}

	// Configuration.
	cfgFilename := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", cfgFilename)
	if err != nil {
		log.Fatalf("Cannot read configuration: %v", err)
	}

}

// EOF
