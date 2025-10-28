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
	minPods := getSliderValueAsInt(controls.sliderHPAMinPods.slider)
	maxPods := getSliderValueAsInt(controls.sliderHPAMaxPods.slider)

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

	// calculate DesiredPods
	desiredPods := float64(currentPods) * cpuMetric / target

	// ceil DesiredPods to next integer using math.Ceil function
	desiredPodsIntRaw := int(math.Ceil(desiredPods))
	desiredPodsInt := desiredPodsIntRaw

	// limit scaling speed according limit function
	maxAllowed := limitScalingSpeed(currentPods)
	if desiredPodsInt > maxAllowed {
		desiredPodsInt = maxAllowed
	}

	// clamp DesiredPods between MinPods and MaxPods
	if desiredPodsInt < minPods {
		desiredPodsInt = minPods
	}
	if desiredPodsInt > maxPods {
		desiredPodsInt = maxPods
	}

	// log inconsistent min vs max
	if minPods > maxPods {
		fmt.Printf("Warning: HPA Min Pods (%d) is greater than HPA Max Pods (%d)\n", minPods, maxPods)
	}

	fmt.Printf("HPA Simulation: currentPods=%d totalCPUUsage=%d podCPURequest=%d cpuMetric=%v targetCPUUtilization=%v => desiredPodsRaw=%d desiredPods=%d\n",
		currentPods, totalCPUUsage, podCPURequest, cpuMetric, target, desiredPodsIntRaw, desiredPodsInt)

	return desiredPodsInt
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
