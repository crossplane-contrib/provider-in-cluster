#!/bin/bash

oc create secret generic cluster-config --from-file=kubeconfig="$1"