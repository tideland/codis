# Tideland CoDis

## Description

**Tideland Configuration Distributor** is a little demo project for the development of Kubernetes operators in Go. Idea is to have a namespace running a controller instance. It listens for `configmaps` and `secrets` and in case they contain a configured label copies them to an also configured list of namespaces. This way it can be used to distribute central configurations and secrets to a number of parallel running namespaces.