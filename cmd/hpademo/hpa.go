package main

import (
	"fmt"
	"math"
)

// runHPADemoSimulation runs a simulation of HPA behavior based on the provided controls.
// HPA formula is:
// DesiredPods = CurrentPods * (cpuMetric / TargetCPUUtilization)
// where cpuMetric = TotalCPUUsage / TotalCPURequest
// and TotalCPURequest = PODCPURequest * CurrentPods
// hence:
// DesiredPods = TotalCPUUsage / (PODCPURequest * CurrentPods * TargetCPUUtilization)
// DesiredPods is ceiled to the next integer if not an integer.
// The result is then clamped between MinPods and MaxPods.
func runHPADemoSimulation(controls podControls) int {
	currentPods := getSliderValueAsInt(controls.sliderNumberOfPods.slider)
	totalCPUUsage := getSliderValueAsInt(controls.sliderCPUUsage.slider)
	podCPULimit := getSliderValueAsInt(controls.sliderPODCPULimit.slider)
	podCPURequest := getSliderValueAsInt(controls.sliderPODCPURequest.slider)
	targetCPUUtilization := getSliderValueAsInt(controls.sliderHPATargetCPUUtilization.slider)
	minReplicas := getSliderValueAsInt(controls.sliderHPAMinReplicas.slider)
	maxReplicas := getSliderValueAsInt(controls.sliderHPAMaxReplicas.slider)

	// calculate totalCPULimit
	totalCPULimit := podCPULimit * currentPods

	// cannot actually load CPU more than the total CPU limit
	if totalCPUUsage > totalCPULimit {
		totalCPUUsage = totalCPULimit
	}

	// calculate totalCPURequest
	totalCPURequest := podCPURequest * currentPods

	// calculate cpuMetric
	cpuMetric := float64(totalCPUUsage) / float64(totalCPURequest)

	target := float64(targetCPUUtilization) / 100

	// calculate currentMetric / desiredMetric
	usageRatio := cpuMetric / target

	var desiredPodsInt int

	// do not scale if within tolerance (cpuMetric close enough to target).
	if withinTolerance(usageRatio) {
		fmt.Printf("hpademo %s: within tolerance (cpuMetric=%v target=%v usageRatio=%v tolerance=%v ratioRange=%v..%v), not scaling\n", version, cpuMetric, target, usageRatio, scaleTolerance, (1.0 - scaleTolerance), (1.0 + scaleTolerance))
		desiredPodsInt = currentPods
	} else {
		// calculate DesiredPods
		desiredPods := float64(currentPods) * usageRatio

		// ceil DesiredPods to next integer using math.Ceil function
		desiredPodsInt = int(math.Ceil(desiredPods))
	}

	// limit scaling speed according limit function
	maxAllowed := limitScalingSpeed(currentPods)
	if desiredPodsInt > maxAllowed {
		desiredPodsInt = maxAllowed
	}

	// clamp DesiredPods between min replicas and max replicas
	if desiredPodsInt < minReplicas {
		desiredPodsInt = minReplicas
	}
	if desiredPodsInt > maxReplicas {
		desiredPodsInt = maxReplicas
	}

	// log inconsistent min vs max
	if minReplicas > maxReplicas {
		fmt.Printf("WARN: HPA Min Replicas (%d) is greater than HPA Max Replicas (%d)\n", minReplicas, maxReplicas)
	}

	fmt.Printf("hpademo %s: currentPods=%d totalCPUUsage=%d podCPURequest=%d cpuMetric=%v targetCPUUtilization=%v => desiredPods=%d\n",
		version, currentPods, totalCPUUsage, podCPURequest, cpuMetric, target, desiredPodsInt)

	return desiredPodsInt
}

const scaleTolerance = 0.1 // 10% for both up and down

// withinTolerance returns true if the usageRatio is within the scale tolerance.
//
// usageRatio = cpuMetric / target
//
// for tolerance=10%, the usageRatio must be between 0.9 and 1.1 to be considered within tolerance.
func withinTolerance(usageRatio float64) bool {
	return usageRatio >= (1.0-scaleTolerance) && usageRatio <= (1.0+scaleTolerance)
}

// limitScalingSpeed limits the scaling speed of the HPA.
//
// see:
//
// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/podautoscaler/horizontal.go
//
//	func calculateScaleUpLimit(currentReplicas int32) int32 {
//		return int32(math.Max(scaleUpLimitFactor*float64(currentReplicas), scaleUpLimitMinimum)) // return max(2*replicas, 4)
//	}
func limitScalingSpeed(currentPods int) int {
	return max(2*currentPods, 4)
}
