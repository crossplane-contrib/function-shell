#!/bin/bash
SA=$(kubectl -n upbound-system get sa -o name | grep function-shell | sed -e 's|serviceaccount\/|upbound-system:|g')
kubectl create clusterrolebinding function-shell-admin-binding --clusterrole cluster-admin --serviceaccount="${SA}"
