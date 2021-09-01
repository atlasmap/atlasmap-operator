package resources

import (
	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ConfigureResources(cr *v1alpha1.AtlasMap, container *corev1.Container) error {
	limits := make(corev1.ResourceList)
	requests := make(corev1.ResourceList)

	if len(cr.Spec.LimitCPU) > 0 {
		cpuLimit, err := resource.ParseQuantity(cr.Spec.LimitCPU)
		if err != nil {
			return err
		}

		limits[corev1.ResourceCPU] = cpuLimit
	}

	if len(cr.Spec.LimitMemory) > 0 {
		memoryLimit, err := resource.ParseQuantity(cr.Spec.LimitMemory)
		if err != nil {
			return err
		}

		limits[corev1.ResourceMemory] = memoryLimit
	}

	if len(cr.Spec.RequestCPU) > 0 {
		cpuRequest, err := resource.ParseQuantity(cr.Spec.RequestCPU)
		if err != nil {
			return err
		}

		requests[corev1.ResourceCPU] = cpuRequest
	}

	if len(cr.Spec.RequestMemory) > 0 {
		memoryRequest, err := resource.ParseQuantity(cr.Spec.RequestMemory)
		if err != nil {
			return err
		}
		requests[corev1.ResourceMemory] = memoryRequest
	}

	container.Resources.Limits = limits
	container.Resources.Requests = requests

	return nil
}

func ResourceListChanged(cr *v1alpha1.AtlasMap, resources corev1.ResourceRequirements) (bool, error) {
	limitsUpdates, err := resourceListQuantityChanged(resources.Limits, cr.Spec.LimitCPU, cr.Spec.LimitMemory)
	if err != nil {
		return false, err
	}

	requestsUpdated, err := resourceListQuantityChanged(resources.Requests, cr.Spec.RequestCPU, cr.Spec.RequestMemory)
	if err != nil {
		return false, err
	}

	return limitsUpdates || requestsUpdated, nil
}

func resourceListQuantityChanged(resourceList corev1.ResourceList, cpu string, memory string) (bool, error) {
	needsUpdate := false
	resources := map[corev1.ResourceName]*resource.Quantity{
		corev1.ResourceCPU:    resourceList.Cpu(),
		corev1.ResourceMemory: resourceList.Memory(),
	}

	for resourceType, resourceValue := range resources {
		newResourceValue, _ := resource.ParseQuantity("0")

		if resourceType == corev1.ResourceCPU {
			if len(cpu) > 0 {
				quantity, err := resource.ParseQuantity(cpu)
				if err != nil {
					return false, err
				}
				newResourceValue = quantity
			}
		}

		if resourceType == corev1.ResourceMemory {
			if len(memory) > 0 {
				quantity, err := resource.ParseQuantity(memory)
				if err != nil {
					return false, err
				}
				newResourceValue = quantity
			}
		}

		if resourceValue.String() != newResourceValue.String() {
			needsUpdate = true
		}
	}
	return needsUpdate, nil
}
