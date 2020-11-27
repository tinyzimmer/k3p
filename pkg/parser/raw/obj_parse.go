package raw

import (
	"github.com/tinyzimmer/k3p/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1betav1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func parseObjectForImages(obj runtime.Object) []string {
	images := make([]string, 0)

	gvk := obj.GetObjectKind().GroupVersionKind()

	switch gvk.Kind {
	case "Pod":
		pod := obj.(*corev1.Pod)
		log.Info("Found Pod:", pod.GetName())
		if imgs := parseImagesFromContainers(pod.Spec.Containers); len(imgs) > 0 {
			images = append(images, imgs...)
		}
	case "Deployment": // only supports appsv1 and v1beta1 for now
		switch gvk.Version {
		case "v1":
			deployment := obj.(*appsv1.Deployment)
			log.Info("Found Deployment:", deployment.GetName())
			if imgs := parseImagesFromContainers(deployment.Spec.Template.Spec.Containers); len(imgs) > 0 {
				images = append(images, imgs...)
			}
		case "v1beta1":
			deployment := obj.(*appsv1beta1.Deployment)
			log.Info("Found Deployment:", deployment.GetName())
			if imgs := parseImagesFromContainers(deployment.Spec.Template.Spec.Containers); len(imgs) > 0 {
				images = append(images, imgs...)
			}
		default:
			log.Info("Skipping non apps/v1 or apps/v1beta1 Deployment object")
		}
	case "StatefulSet": // only supports apps/v1
		ss, ok := obj.(*appsv1.StatefulSet)
		if !ok {
			log.Info("Skipping non apps/v1 StatefulSet object")
			return images
		}
		log.Info("Found StatefulSet:", ss.GetName())
		if imgs := parseImagesFromContainers(ss.Spec.Template.Spec.Containers); len(imgs) > 0 {
			images = append(images, imgs...)
		}
	case "Job": // only supports batch/v1
		job, ok := obj.(*batchv1.Job)
		if !ok {
			log.Info("Skipping non batch/v1 Job object")
			return images
		}
		log.Info("Found Job:", job.GetName())
		if imgs := parseImagesFromContainers(job.Spec.Template.Spec.Containers); len(imgs) > 0 {
			images = append(images, imgs...)
		}
	case "CronJob": // only supports batch/v1beta1
		job, ok := obj.(*batchv1betav1.CronJob)
		if !ok {
			log.Info("Skipping non batch/v1betav1 CronJob object")
			return images
		}
		log.Info("Found CronJob:", job.GetName())
		if imgs := parseImagesFromContainers(job.Spec.JobTemplate.Spec.Template.Spec.Containers); len(imgs) > 0 {
			images = append(images, imgs...)
		}
	default:
		log.Debug("Skipping non-container based object:", gvk.Kind) // TODO: verbose logging
	}

	return images
}

func parseImagesFromContainers(containers []corev1.Container) []string {
	images := make([]string, 0)
	for _, container := range containers {
		if container.Image != "" {
			images = append(images, container.Image)
		}
	}
	return images
}
