package v1beta1

import (
	"crypto/md5"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func hashString(s string) string {
	return fmt.Sprintf("%x\n", md5.Sum([]byte(s)))
}

func (f *Function) GetServerPort32() int32 {
	if f.Spec.ServerPort != 0 {
		return int32(f.Spec.ServerPort)
	}

	return int32(8080)
}

func (f *Function) Deployment(name, namespace string) appsv1.Deployment {
	var replicas = int32(f.Spec.Replicas)

	if f.Spec.Replicas == 0 {
		replicas = int32(1)
	}

	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:            "builder",
							Image:           "ghcr.io/hooksie1/goless-builder",
							ImagePullPolicy: "IfNotPresent",
							VolumeMounts: []corev1.VolumeMount{
								{Name: "build", MountPath: "/server"},
								{Name: "handler", MountPath: "/handlers"},
							},
						},
					},
					Containers: []corev1.Container{{
						Name:            "server",
						Image:           "ghcr.io/hooksie1/goless-server",
						ImagePullPolicy: "IfNotPresent",
						Env: []corev1.EnvVar{
							{
								Name:  "SERVER_PORT",
								Value: fmt.Sprintf("%d", f.GetServerPort32()),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "build", MountPath: "/server"},
						},
					}},
					Volumes: []corev1.Volume{
						{Name: "handler",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name,
									},
								},
							},
						},
						{Name: "build"},
					},
				},
			},
		},
	}
}

func (f *Function) ConfigMap(name, namespace string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"hash": hashString(f.Spec.Function),
			},
		},
		Data: map[string]string{
			"handler.go": f.Spec.Function,
		},
	}
}

func (f *Function) Service(name, namespace string) corev1.Service {
	return corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"hash": hashString(f.Spec.Function),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       f.GetServerPort32(),
					TargetPort: intstr.FromInt(int(f.GetServerPort32())),
				},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}
}
