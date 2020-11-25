module github.com/layer5io/meshery-operator

go 1.13

replace (
	github.com/kudobuilder/kuttl => github.com/layer5io/kuttl v0.4.1-0.20200806180306-b7e46afd657f
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)

require (
	cloud.google.com/go v0.62.0 // indirect
	github.com/allegro/bigcache v1.2.1
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/layer5io/meshkit v0.1.27
	github.com/myntra/pipeline v0.0.0-20180618182531-2babf4864ce8
	github.com/nats-io/nats.go v1.10.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	golang.org/x/tools v0.0.0-20200804011535-6c149bb5ef0d // indirect
	google.golang.org/genproto v0.0.0-20200804131852-c06518451d9c // indirect
	google.golang.org/grpc v1.31.0
	google.golang.org/protobuf v1.25.0
	istio.io/client-go v1.8.0-alpha.2
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.18.12
	k8s.io/client-go v0.18.12
	sigs.k8s.io/controller-runtime v0.6.2
)
