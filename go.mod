module github.com/layer5io/meshery-operator

go 1.13

replace (
	github.com/kudobuilder/kuttl => github.com/layer5io/kuttl v0.4.1-0.20200806180306-b7e46afd657f
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
// golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6
)

require (
	cloud.google.com/go v0.62.0 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/layer5io/meshkit v0.2.6
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	golang.org/x/tools v0.0.0-20200804011535-6c149bb5ef0d // indirect
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.18.12
	k8s.io/client-go v0.18.12
	sigs.k8s.io/controller-runtime v0.6.4
)
