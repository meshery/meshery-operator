package controllers

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyncBrokerStatefulSet(t *testing.T) {
	const newAnnotation = "new"
	one := int32(1)
	two := int32(2)

	existing := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": "old"},
			Annotations: map[string]string{"team": "old"},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &one,
			ServiceName: "old",
		},
	}
	desired := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": newAnnotation},
			Annotations: map[string]string{"team": newAnnotation},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &two,
			ServiceName: newAnnotation,
		},
	}

	if changed := syncBrokerStatefulSet(existing, desired); !changed {
		t.Fatal("expected statefulset sync to report changes")
	}
	if existing.Labels["app"] != newAnnotation {
		t.Fatalf("expected labels to be updated, got %v", existing.Labels)
	}
	if existing.Annotations["team"] != newAnnotation {
		t.Fatalf("expected annotations to be updated, got %v", existing.Annotations)
	}
	if existing.Spec.Replicas == nil || *existing.Spec.Replicas != two {
		t.Fatalf("expected replicas to be updated to %d, got %v", two, existing.Spec.Replicas)
	}
	if existing.Spec.ServiceName != newAnnotation {
		t.Fatalf("expected service name to be updated, got %q", existing.Spec.ServiceName)
	}
	if changed := syncBrokerStatefulSet(existing, desired); changed {
		t.Fatal("expected no-op statefulset sync after objects match")
	}
}

func TestSyncBrokerService(t *testing.T) {
	const newAnnotation = "new"
	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": "old"},
			Annotations: map[string]string{"team": "old"},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": "old"},
			Ports: []corev1.ServicePort{
				{Name: "client", Port: 4222},
			},
			ClusterIP: "10.0.0.1",
		},
	}
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": newAnnotation},
			Annotations: map[string]string{"team": newAnnotation},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{"app": newAnnotation},
			Ports: []corev1.ServicePort{
				{Name: "client", Port: 5222},
			},
		},
	}

	if changed := syncBrokerService(existing, desired); !changed {
		t.Fatal("expected service sync to report changes")
	}
	if existing.Labels["app"] != newAnnotation {
		t.Fatalf("expected labels to be updated, got %v", existing.Labels)
	}
	if existing.Annotations["team"] != newAnnotation {
		t.Fatalf("expected annotations to be updated, got %v", existing.Annotations)
	}
	if existing.Spec.Type != corev1.ServiceTypeLoadBalancer {
		t.Fatalf("expected service type to be updated, got %s", existing.Spec.Type)
	}
	if existing.Spec.Ports[0].Port != 5222 {
		t.Fatalf("expected service port to be updated, got %d", existing.Spec.Ports[0].Port)
	}
	if existing.Spec.Selector["app"] != newAnnotation {
		t.Fatalf("expected selector to be updated, got %v", existing.Spec.Selector)
	}
	if existing.Spec.ClusterIP != "10.0.0.1" {
		t.Fatalf("expected clusterIP to remain untouched, got %q", existing.Spec.ClusterIP)
	}
	if changed := syncBrokerService(existing, desired); changed {
		t.Fatal("expected no-op service sync after objects match")
	}
}

func TestSyncBrokerConfigMap(t *testing.T) {
	const newAnnotation = "new"
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": "old"},
			Annotations: map[string]string{"team": "old"},
		},
		Data:       map[string]string{"config": "old"},
		BinaryData: map[string][]byte{"bin": []byte("old")},
	}
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": newAnnotation},
			Annotations: map[string]string{"team": newAnnotation},
		},
		Data:       map[string]string{"config": newAnnotation},
		BinaryData: map[string][]byte{"bin": []byte(newAnnotation)},
	}

	if changed := syncBrokerConfigMap(existing, desired); !changed {
		t.Fatal("expected configmap sync to report changes")
	}
	if existing.Data["config"] != newAnnotation {
		t.Fatalf("expected configmap data to be updated, got %v", existing.Data)
	}
	if string(existing.BinaryData["bin"]) != newAnnotation {
		t.Fatalf("expected configmap binary data to be updated, got %q", string(existing.BinaryData["bin"]))
	}
	if changed := syncBrokerConfigMap(existing, desired); changed {
		t.Fatal("expected no-op configmap sync after objects match")
	}
}

func TestSyncMeshsyncDeployment(t *testing.T) {
	const newAnnotation = "new"
	one := int32(1)
	two := int32(2)

	existing := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": "old"},
			Annotations: map[string]string{"team": "old"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
		},
	}
	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"app": newAnnotation},
			Annotations: map[string]string{"team": newAnnotation},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &two,
		},
	}

	if changed := syncMeshsyncDeployment(existing, desired); !changed {
		t.Fatal("expected deployment sync to report changes")
	}
	if existing.Labels["app"] != newAnnotation {
		t.Fatalf("expected labels to be updated, got %v", existing.Labels)
	}
	if existing.Annotations["team"] != newAnnotation {
		t.Fatalf("expected annotations to be updated, got %v", existing.Annotations)
	}
	if existing.Spec.Replicas == nil || *existing.Spec.Replicas != two {
		t.Fatalf("expected replicas to be updated to %d, got %v", two, existing.Spec.Replicas)
	}
	if changed := syncMeshsyncDeployment(existing, desired); changed {
		t.Fatal("expected no-op deployment sync after objects match")
	}
}
