module github.com/layer5io/meshery-operator

go 1.17

replace (
	github.com/kudobuilder/kuttl => github.com/layer5io/kuttl v0.4.1-0.20200806180306-b7e46afd657f
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
//	golang.org/x/sys => golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.8.1
	k8s.io/api => k8s.io/api v0.22.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.8
	k8s.io/client-go => k8s.io/client-go v0.22.8
	k8s.io/kubectl => k8s.io/kubectl v0.22.8
)

require (
	github.com/go-logr/logr v0.4.0
	github.com/layer5io/meshkit v0.5.16
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	k8s.io/api v0.23.0-alpha.1
	k8s.io/apimachinery v0.23.0-alpha.1
	k8s.io/client-go v0.23.0-alpha.1
	sigs.k8s.io/controller-runtime v0.11.2
)
