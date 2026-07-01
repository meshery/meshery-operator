package broker

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const (
	mesheryName       = "meshery"
	appLabelKey       = "app"
	componentLabelKey = "component"
	brokerComponent   = "broker"
	natsServiceName   = "meshery-nats"
	clientPortName    = "client"
	monitorPortName   = "monitor"
	configVolumeName  = "config-volume"
	pidVolumeName     = "pid"
	resolverVolume    = "resolver-volume"

	// defaultNATSVersion is the NATS server image tag used when BrokerSpec.Version
	// is empty. Override per-Broker via spec.version.
	defaultNATSVersion = "2.10.29-alpine3.21"
	// natsConfigReloaderImage is the maintained config-reloader (the former
	// connecteverything org is defunct and now lives under natsio).
	natsConfigReloaderImage = "natsio/nats-server-config-reloader:0.23.0"
)

// natsImage returns the NATS server image for the given spec version, falling
// back to the operator's bundled default.
func natsImage(version string) string {
	if version == "" {
		version = defaultNATSVersion
	}
	return "nats:" + version
}

var (
	val1    int32 = 1
	val60   int64 = 60
	val4222 int32 = 4222
	val6222 int32 = 6222
	val7422 int32 = 7422
	val7522 int32 = 7522
	val8222 int32 = 8222
	val7777 int32 = 7777

	valtrue bool = true

	MesheryLabel = map[string]string{
		appLabelKey: mesheryName,
	}

	MesheryAnnotation = map[string]string{
		"meshery/component-type": "management-plane",
	}

	BrokerLabel = map[string]string{
		appLabelKey:       MesheryLabel[appLabelKey],
		componentLabelKey: brokerComponent,
	}

	PrometheusAnnotation = map[string]string{
		"meshery/component-type": "management-plane",
		"prometheus.io/path":     "/metrics",
		"prometheus.io/port":     "7777",
		"prometheus.io/scrape":   "true",
	}

	NatsConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mesheryName,
			Name:      "meshery-nats-config",
			Labels:    BrokerLabel,
		},
		Data: map[string]string{
			"nats.conf": `
# PID file shared with configuration reloader.
pid_file: "/var/run/nats/nats.pid"
# Monitoring
http: 8222
server_name: $POD_NAME
# Authorization 
resolver: MEMORY
include "accounts/resolver.conf"`,
		},
	}

	AccountsConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mesheryName,
			Name:      "meshery-nats-accounts",
			Labels:    BrokerLabel,
		},
		Data: map[string]string{
			"resolver.conf": `
resolver: MEMORY
resolver_preload: {
ACSU3Q6LTLBVLGAQUONAGXJHVNWGSKKAUA7IY5TB4Z7PLEKSR5O6JTGR: eyJ0eXAiOiJqd3QiLCJhbGciOiJlZDI1NTE5In0.eyJqdGkiOiJPRFhJSVI2Wlg1Q1AzMlFJTFczWFBENEtTSDYzUFNNSEZHUkpaT05DR1RLVVBISlRLQ0JBIiwiaWF0IjoxNTU2NjU1Njk0LCJpc3MiOiJPRFdaSjJLQVBGNzZXT1dNUENKRjZCWTRRSVBMVFVJWTRKSUJMVTRLM1lERzNHSElXQlZXQkhVWiIsIm5hbWUiOiJBIiwic3ViIjoiQUNTVTNRNkxUTEJWTEdBUVVPTkFHWEpIVk5XR1NLS0FVQTdJWTVUQjRaN1BMRUtTUjVPNkpUR1IiLCJ0eXBlIjoiYWNjb3VudCIsIm5hdHMiOnsibGltaXRzIjp7InN1YnMiOi0xLCJjb25uIjotMSwibGVhZiI6LTEsImltcG9ydHMiOi0xLCJleHBvcnRzIjotMSwiZGF0YSI6LTEsInBheWxvYWQiOi0xLCJ3aWxkY2FyZHMiOnRydWV9fX0._WW5C1triCh8a4jhyBxEZZP8RJ17pINS8qLzz-01o6zbz1uZfTOJGvwSTS6Yv2_849B9iUXSd-8kp1iMXHdoBA
}`,
		},
	}

	Service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   mesheryName,
			Name:        natsServiceName,
			Labels:      BrokerLabel,
			Annotations: MesheryAnnotation,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: clientPortName,
					Port: val4222,
				},
				{
					Name: "cluster",
					Port: val6222,
				},
				{
					Name: monitorPortName,
					Port: val8222,
				},
				{
					Name: "metrics",
					Port: val7777,
				},
				{
					Name: "leafnodes",
					Port: val7422,
				},
				{
					Name: "gateways",
					Port: val7522,
				},
			},
			Selector: BrokerLabel,
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}

	StatefulSet = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   mesheryName,
			Name:        natsServiceName,
			Labels:      BrokerLabel,
			Annotations: MesheryAnnotation,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &val1,
			Selector: &metav1.LabelSelector{
				MatchLabels: BrokerLabel,
			},
			ServiceName: natsServiceName,
			Template:    PodTemplate,
		},
	}

	PodTemplate = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   mesheryName,
			Name:        natsServiceName,
			Labels:      BrokerLabel,
			Annotations: PrometheusAnnotation,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "meshery-operator",
			Volumes: []corev1.Volume{
				{
					Name: configVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "meshery-nats-config",
							},
						},
					},
				},
				{
					Name: pidVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: resolverVolume,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "meshery-nats-accounts",
							},
						},
					},
				},
			},
			ShareProcessNamespace:         &valtrue,
			TerminationGracePeriodSeconds: &val60,
			Containers: []corev1.Container{
				{
					Name:            "nats",
					Image:           "nats:" + defaultNATSVersion,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports: []corev1.ContainerPort{
						{
							Name: clientPortName,

							ContainerPort: val4222,
						},
						{
							Name: "cluster",

							ContainerPort: val6222,
						},
						{
							Name: "leafnodes",

							ContainerPort: val7422,
						},
						{
							Name: "gateways",

							ContainerPort: val7522,
						},
						{
							Name: monitorPortName,

							ContainerPort: val8222,
						},
						{
							Name:          "metrics",
							ContainerPort: val7777,
						},
					},
					Command: []string{
						"nats-server", "--config", "/etc/nats-config/nats.conf",
					},
					Env: []corev1.EnvVar{
						{
							Name: "POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
						{
							Name:  "CLUSTER_ADVERTISE",
							Value: "$(POD_NAME).meshery-nats.$(POD_NAMESPACE).svc",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: "/etc/nats-config",
						},
						{
							Name:      pidVolumeName,
							MountPath: "/var/run/nats",
						},
						{
							Name:      resolverVolume,
							MountPath: "/etc/nats-config/accounts",
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/",
								Port: intstr.IntOrString{
									IntVal: val8222,
								},
							},
						},
						InitialDelaySeconds: 10,
						TimeoutSeconds:      5,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/",
								Port: intstr.IntOrString{
									IntVal: val8222,
								},
							},
						},
						InitialDelaySeconds: 10,
						TimeoutSeconds:      5,
					},
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/bin/sh", "-c", "nats-server -sl=ldm=/var/run/nats/nats.pid && /bin/sleep 60",
								},
							},
						},
					},
				},
				{
					Name:            "reloader",
					Image:           natsConfigReloaderImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"nats-server-config-reloader", "-pid", "/var/run/nats/nats.pid", "-config", "/etc/nats-config/nats.conf",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: "/etc/nats-config",
						},
						{
							Name:      pidVolumeName,
							MountPath: "/var/run/nats",
						},
						// The natsio reloader (unlike the defunct connecteverything
						// one) follows `include` directives in nats.conf, so it must
						// also see the accounts config that nats.conf includes;
						// otherwise it exits with "accounts/resolver.conf does not
						// exist" and crash-loops the pod.
						{
							Name:      resolverVolume,
							MountPath: "/etc/nats-config/accounts",
						},
					},
				},
			},
		},
	}
)
