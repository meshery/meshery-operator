/*
Copyright Meshery Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package meshsync

import (
	"github.com/meshery/meshery-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	appLabelKey     = "app"
	componentKey    = "component"
	mesheryName     = "meshery"
	meshsyncBinary  = "./meshery-meshsync"
	meshsyncName    = "meshery-meshsync"
	meshsyncService = "meshsync"

	// clientPortName names the container port MeshSync listens on. It carries
	// the broker client traffic and, as of v1.0.1, also serves the /healthz and
	// /readyz HTTP health endpoints, so the httpGet probes target it by name.
	clientPortName = "client"

	// meshsyncImageRepo is the image repository spec.version tags resolve
	// against.
	meshsyncImageRepo = "meshery/meshsync"
	// defaultMeshSyncVersion is the image tag used when spec.version is empty.
	// It MUST be a pinned release that exists on Docker Hub, never a moving
	// channel tag: a moving default re-points under running clusters, so a
	// Meshery/chart combination that installed cleanly starts pulling an
	// incompatible MeshSync on the next pod restart (the operator's own
	// stable-latest did exactly this - see docs/release-process.md).
	// Registry convention is bare semver (meshery/meshsync:1.0.3); the
	// v-prefixed form is a git/release tag and does not exist as an image tag.
	//
	// Bump deliberately: `.github/workflows/meshsync-version-bump.yml` opens a
	// PR when meshery/meshsync publishes a newer release, and
	// TestDefaultMeshSyncVersionIsPinned fails if this ever regresses to a
	// moving tag.
	defaultMeshSyncVersion = "1.0.3"
)

var (
	val1     int32 = 1
	val60    int64 = 60
	val11000 int32 = 11000

	valtrue bool = true

	MesheryLabel = map[string]string{
		appLabelKey: mesheryName,
	}

	MeshSyncLabel = map[string]string{
		appLabelKey:  MesheryLabel[appLabelKey],
		componentKey: meshsyncService,
	}

	MesheryAnnotation = map[string]string{
		"meshery/component-type": "management-plane",
	}

	// Resource limits and requests
	CPURequest    = resource.MustParse("500m")
	CPULimit      = resource.MustParse("4")
	MemoryRequest = resource.MustParse("512Mi")
	MemoryLimit   = resource.MustParse("4Gi")

	Deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        meshsyncName,
			Labels:      MeshSyncLabel,
			Annotations: MesheryAnnotation,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &val1,
			Selector: &metav1.LabelSelector{
				MatchLabels: MeshSyncLabel,
			},
			Template: PodTemplate,
		},
	}

	PodTemplate = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:        meshsyncName,
			Labels:      MeshSyncLabel,
			Annotations: MesheryAnnotation,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName:            "meshery-operator",
			ShareProcessNamespace:         &valtrue,
			TerminationGracePeriodSeconds: &val60,
			Containers: []corev1.Container{
				{
					Name: "meshsync",
					// Image and ImagePullPolicy are authoritative only as the
					// starting point: GetServerObject runs applyVersion over
					// every built Deployment, so both always end up derived
					// from the resolved version (spec.version, or the pinned
					// default above). They are seeded consistently here so the
					// template alone is never a moving-tag reference.
					Image:           meshsyncImageRepo + ":" + defaultMeshSyncVersion,
					ImagePullPolicy: utils.PullPolicyFor(defaultMeshSyncVersion),
					Ports: []corev1.ContainerPort{
						{
							// No HostPort: pinning a host port would limit
							// scheduling to one MeshSync per node for a port
							// nothing external needs to reach.
							Name:          clientPortName,
							ContainerPort: val11000,
						},
					},
					Command: []string{
						meshsyncBinary,
					},
					Env: []corev1.EnvVar{
						{
							Name:  brokerURLEnv,
							Value: defaultBrokerURL,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    CPURequest,
							corev1.ResourceMemory: MemoryRequest,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    CPULimit,
							corev1.ResourceMemory: MemoryLimit,
						},
					},
					// Exec liveness is the version-skew-safe fallback baked into
					// the template. As of v1.0.1 MeshSync serves HTTP /healthz
					// (always 200) and /readyz (503 until it has connected to the
					// broker once, then a permanent 200) on the client port, and
					// applyProbes upgrades pods whose resolved version is a pinned
					// semver >= that release to httpGet probes - which now includes
					// the default, since defaultMeshSyncVersion is itself pinned.
					// Everything else keeps this exec probe: spec.version is
					// user-settable, and a moving or unparseable tag can't be
					// proven to carry the endpoints, so an httpGet probe against an
					// image that serves nothing on the client port would
					// connection-refuse and crashloop an otherwise-healthy pod.
					// `-h` only proves the binary can fork - nothing about sync
					// health - and that fork can exceed a tight timeout during the
					// initial full-cluster sync on CPU-starved nodes, so the probe
					// stays deliberately relaxed. No readiness probe is attached
					// for the same version-skew reason (and no Service routes to
					// MeshSync, so readiness would gate no traffic); applyProbes
					// adds one only for versions known to serve /readyz.
					LivenessProbe: &corev1.Probe{
						InitialDelaySeconds: 60,
						PeriodSeconds:       30,
						TimeoutSeconds:      30,
						FailureThreshold:    5,
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									meshsyncBinary,
									"-h",
								},
							},
						},
					},
				},
			},
		},
	}
)
