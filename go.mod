module github.com/layer5io/meshery-operator

go 1.13

replace (
	github.com/kudobuilder/kuttl => github.com/layer5io/kuttl v0.4.1-0.20200806180306-b7e46afd657f
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)

require (
	github.com/bombsimon/logrusr v1.0.0
	github.com/go-logr/logr v0.3.0
	github.com/layer5io/meshkit v0.1.31
	github.com/layer5io/meshsync v0.0.0-20201217151817-9cddb43166ca
	github.com/nats-io/nats.go v1.10.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/sirupsen/logrus v1.6.0
	istio.io/client-go v1.8.1
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.18.12
	k8s.io/client-go v0.18.12
	sigs.k8s.io/controller-runtime v0.6.4
)
