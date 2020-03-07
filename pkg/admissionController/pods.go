package admissioncontroller

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/AlexisDuf/k8sWebhooks/pkg/config"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	podsInitContainerPatch string = `[
		{"op":"add","path":"/spec/initContainers","value":[{"image":"webhook-added-image","name":"webhook-added-init-container","resources":{}}]}
	]`
	podsSidecarPatch string = `[
		{"op":"add", "path":"/spec/containers/-","value":{"image":"%v","name":"webhook-added-sidecar","resources":{}}}
	]`

	sidecarImageAnnotInjectKey = "sidecar-image/inject"
	sidecarImageAnnotStatusKey = "sidecar-image/status"
)

func createPodPatch()

func mutatePodsSidecar(ar v1.AdmissionReview, sidecar *config.Sidecar) *v1.AdmissionResponse {
	shouldPatchPod := func(pod *corev1.Pod) bool {
		return hasAnnotation(&pod.ObjectMeta)
	}

	return applyPodPatch(ar, shouldPatchPod)
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

func applyPodPatch(ar v1.AdmissionReview, shouldPatchPod func(*corev1.Pod) bool) *v1.AdmissionResponse {
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
	if shouldPatchPod(&pod) {
		var mutationOperation []PatchOperation

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
