package admissioncontroller

import (
	"encoding/json"
	"log"
	"strings"

	config "github.com/AlexisDuf/k8sWebhooks/pkg/config"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	sidecarImageAnnotInjectKey = "sidecar-image/inject"
	sidecarImageAnnotStatusKey = "sidecar-image/status"
)

func addContainer(target, added []corev1.Container, basePath string) (patch []PatchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, PatchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func updateAnnotations(target map[string]string, added map[string]string) (patch []PatchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, PatchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, PatchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}

	log.Printf("%v", patch)
	return patch
}

func createPatch(pod *corev1.Pod, config *config.Sidecar, annotations map[string]string) ([]PatchOperation, error) {
	var patch []PatchOperation

	patch = append(patch, addContainer(pod.Spec.Containers, config.Containers, "/spec/containers")...)
	patch = append(patch, updateAnnotations(pod.Annotations, annotations)...)

	log.Printf("%v", patch)
	return patch, nil
}

func hasAnnotation(metadata *metav1.ObjectMeta) bool {
	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status := annotations[sidecarImageAnnotStatusKey]

	var required bool
	if strings.ToLower(status) == "injected" {
		required = false
	} else {
		switch strings.ToLower(annotations[sidecarImageAnnotInjectKey]) {
		default:
			required = false
		case "y", "yes", "true", "on":
			required = true
		}
	}
	log.Printf("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)

	return required
}

func applyPodPatch(ar v1.AdmissionReview, config *config.Sidecar) *v1.AdmissionResponse {
	klog.V(2).Info("Mutating pods")
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		klog.Errorf("expect resource to be %s", podResource)
		return nil
	}

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		klog.Error(err)
		return toV1AdmissionResponse(err)
	}
	reviewResponse := v1.AdmissionResponse{}
	reviewResponse.Allowed = true
	if hasAnnotation(&pod.ObjectMeta) {
		annotations := map[string]string{sidecarImageAnnotStatusKey: "injected"}

		mutationOperation, err := createPatch(&pod, config, annotations)
		if err != nil {
			klog.Error(err)
		}

		patch, err := json.Marshal(mutationOperation)

		if err != nil {
			klog.Error(err)
		}

		reviewResponse.Patch = patch
		pt := v1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt
	}
	return &reviewResponse
}
