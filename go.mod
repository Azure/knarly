module github.com/azure/knarly

go 1.16

require (
	github.com/Azure/aad-pod-identity v1.8.5
	github.com/Azure/azure-sdk-for-go v58.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.21
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/drone/envsubst/v2 v2.0.0-20210730161058-179042472c46 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/mod v0.5.1
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985 // indirect
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.0.2
	sigs.k8s.io/cluster-api-provider-azure v1.1.1-0.20220113202229-e123e78fcee9
	sigs.k8s.io/cluster-api/test v1.0.2
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/kind v0.11.1
	sigs.k8s.io/kustomize/kustomize/v4 v4.2.0 // indirect
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.0.2
