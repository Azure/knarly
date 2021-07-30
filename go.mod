module github.com/azure/knarly

go 1.16

require (
	github.com/Azure/aad-pod-identity v1.8.2
	github.com/Azure/azure-sdk-for-go v56.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.3
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/drone/envsubst/v2 v2.0.0-20210730161058-179042472c46 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/pkg/errors v0.9.1
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/cluster-api v0.4.0
	sigs.k8s.io/cluster-api-provider-azure v0.5.1
	sigs.k8s.io/cluster-api/test v0.4.0
	sigs.k8s.io/controller-runtime v0.9.1
	sigs.k8s.io/kind v0.11.1
	sigs.k8s.io/kustomize/kustomize/v4 v4.2.0 // indirect
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.4.0
