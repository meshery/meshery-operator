module github.com/layer5io/meshery-operator

go 1.13

require (
	github.com/Azure/go-autorest/autorest/adal v0.9.0 // indirect
	github.com/Sirupsen/logrus v1.6.0 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	istio.io/istio v0.0.0-20200723145350-f865b0104ef1
	istio.io/pkg v0.0.0-20200722144425-ffe8ce8a2896
	k8s.io/client-go v0.18.3
)

replace github.com/Sirupsen/logrus v1.6.0 => github.com/sirupsen/logrus v1.6.0
