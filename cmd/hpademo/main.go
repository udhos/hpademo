// Package main implements the tool.
package main

import (
	"fmt"
	"math"
	"strconv"
	"syscall/js"
	"time"
)

type subchart struct {
	ctx    js.Value
	legend js.Value
	data   []int
}

type chart struct {
	pods         subchart
	podsStarting subchart
	podsStopping subchart
	podsLoad     subchart
	unmetLoad    subchart
	canvasWidth  int
	canvasHeight int
}

func getSliderValueAsInt(slider js.Value) int {
	s := slider.Get("value").String()
	i, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("Error converting slider value to int: %v\n", err)
		return 0
	}
	return i
}

func main() {
	document := js.Global().Get("document")

	// add title with version
	titleVersion := fmt.Sprintf("🚀 HPA Demo v%s", version)
	titleElement := document.Call("getElementById", "title")
	titleElement.Set("innerHTML", titleVersion)

	// find canvasPods element
	canvasPods := document.Call("getElementById", "canvas_pods")
	canvasPodsLegend := document.Call("getElementById", "canvas_pods_legend")
	canvasPodsCtx := canvasPods.Call("getContext", "2d")

	canvasPodsLoad := document.Call("getElementById", "canvas_pod_cpu_usage")
	canvasPodsLoadLegend := document.Call("getElementById", "canvas_pod_cpu_usage_legend")
	canvasPodsLoadCtx := canvasPodsLoad.Call("getContext", "2d")

	canvasUnmetLoad := document.Call("getElementById", "canvas_unmet_cpu_load")
	canvasUnmetLoadLegend := document.Call("getElementById", "canvas_unmet_cpu_load_legend")
	canvasUnmetLoadCtx := canvasUnmetLoad.Call("getContext", "2d")

	deploy := deployment{
		desiredReplicas: 1,
	}

	// get canvas width and height
	canvasWidth := canvasPods.Get("width").Int()
	canvasHeight := canvasPods.Get("height").Int()

	const historySize = 600

	c := newChart(canvasPodsCtx, canvasPodsLoadCtx, canvasUnmetLoadCtx,
		canvasPodsLegend, canvasPodsLoadLegend, canvasUnmetLoadLegend,
		canvasWidth, canvasHeight, historySize)

	controls := addHTMLControls(document, func(value string) {
		// Update history size based on slider input
		historySize, err := strconv.Atoi(value)
		if err != nil {
			fmt.Printf("Error converting history size to int: %v\n", err)
			return
		}
		c.resizeHistory(historySize)
	})

	// call function to draw chart
	drawCharts(canvasPodsCtx, canvasPodsLoadCtx, canvasUnmetLoadCtx, c)

	var lastHPAEvaluation int
	var lastScanlingDown time.Time

	// call updateChart every second
	js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) any {
		//
		// evaluate hpa
		//
		var newPodValue int
		lastHPAEvaluation++
		if lastHPAEvaluation >= 15 {
			// get from HPA simulation
			lastHPAEvaluation = 0
			oldPodValue := getSliderValueAsInt(controls.sliderNumberOfPods.slider)

			var isScaleToleranceAllowed bool
			newPodValue, isScaleToleranceAllowed = runHPADemoSimulation(controls)

			isScaling := newPodValue != oldPodValue

			var isScalingDownAllowed bool

			isScalingDown := newPodValue < oldPodValue
			if isScalingDown {
				// scaling down, check stabilization window
				elapSecs := time.Since(lastScanlingDown).Seconds()
				scaleDownWindow := float64(getSliderValueAsInt(controls.sliderScaleDownStabilizationWindow.slider))
				isScalingDownAllowed = elapSecs > scaleDownWindow
				if isScalingDownAllowed {
					lastScanlingDown = time.Now()
				} else {
					fmt.Printf("lastScaleDown=%v <= scaleDownStabilizationWindow=%v, not scaling down\n",
						elapSecs, scaleDownWindow)
				}
			}

			// isScaleToleranceAllowed
			// isScalingDownAllowed
			// isScaling
			// isScalingDown
			willScale := true
			if !isScaleToleranceAllowed {
				willScale = false // do not scale because ratio is within Tolerance range
			}
			if !isScaling {
				willScale = false // do not scale because pods unchanged
			}
			if isScalingDown && !isScalingDownAllowed {
				// do not scale because cannot scale down within stabilization window
				willScale = false
			}

			if willScale {
				// update number of pods slider to reflect HPA decision
				controls.sliderNumberOfPods.slider.Set("value", newPodValue)
				controls.sliderNumberOfPods.textBox.Set("value", newPodValue)
			} else {
				newPodValue = oldPodValue // revert scale
			}
		} else {
			// get from slider
			newPodValue = getSliderValueAsInt(controls.sliderNumberOfPods.slider)
		}

		deploy.startupTime = time.Second * time.Duration(getSliderValueAsInt(controls.sliderPODStartupTime.slider))
		deploy.stopTime = time.Second * time.Duration(getSliderValueAsInt(controls.sliderPODStopTime.slider))

		deploy.scale(newPodValue)

		deploy.update()

		//
		// evaluate per pod load
		//

		currentPods := float64(deploy.getReplicas())
		totalCPUUsage := float64(getSliderValueAsInt(controls.sliderCPUUsage.slider))
		podCPULimit := float64(getSliderValueAsInt(controls.sliderPODCPULimit.slider))

		newPodLoad := totalCPUUsage / currentPods
		newPodLoad = min(newPodLoad, podCPULimit)

		//
		// evaluate total unmet load
		//

		metLoad := newPodLoad * currentPods
		newUnmetLoad := totalCPUUsage - metLoad

		// update chart data
		updateChart(&c,
			deploy.getReplicas(), deploy.getStarting(), deploy.getStopping(),
			int(newPodLoad), int(newUnmetLoad))

		// redraw chart
		drawCharts(canvasPodsCtx, canvasPodsLoadCtx, canvasUnmetLoadCtx, c)

		return nil
	}), 1000)

	// prevent main from exiting

	fmt.Println("waiting forever...")
	select {}
}

type podControls struct {
	sliderCPUUsage                     sliderControl
	sliderPODCPURequest                sliderControl
	sliderPODCPULimit                  sliderControl
	sliderHPAMinReplicas               sliderControl
	sliderHPAMaxReplicas               sliderControl
	sliderHPATargetCPUUtilization      sliderControl
	sliderNumberOfPods                 sliderControl
	sliderHistorySize                  sliderControl
	sliderScaleDownStabilizationWindow sliderControl
	sliderPODStartupTime               sliderControl
	sliderPODStopTime                  sliderControl
}

type sliderControl struct {
	slider  js.Value
	textBox js.Value
}

func addHTMLControls(document js.Value, callbackHistorySize func(string)) podControls {

	var controls podControls

	// Get references to existing HTML elements by ID
	controls.sliderCPUUsage = getSliderControl(document, "slider-cpu-usage", "textbox-cpu-usage")
	controls.sliderPODCPURequest = getSliderControl(document, "slider-pod-cpu-request", "textbox-pod-cpu-request")
	controls.sliderPODCPULimit = getSliderControl(document, "slider-pod-cpu-limit", "textbox-pod-cpu-limit")
	controls.sliderHPAMinReplicas = getSliderControl(document, "slider-hpa-min-replicas", "textbox-hpa-min-replicas")
	controls.sliderHPAMaxReplicas = getSliderControl(document, "slider-hpa-max-replicas", "textbox-hpa-max-replicas")
	controls.sliderHPATargetCPUUtilization = getSliderControl(document, "slider-hpa-target-cpu", "textbox-hpa-target-cpu")
	controls.sliderNumberOfPods = getSliderControl(document, "slider-number-of-pods", "textbox-number-of-pods")
	controls.sliderHistorySize = getSliderControl(document, "slider-history-size", "textbox-history-size")
	controls.sliderScaleDownStabilizationWindow = getSliderControl(document, "slider-scale-down-stabilization-window", "textbox-scale-down-stabilization-window")
	controls.sliderPODStartupTime = getSliderControl(document, "slider-pod-startup-time", "textbox-pod-startup-time")
	controls.sliderPODStopTime = getSliderControl(document, "slider-pod-stop-time", "textbox-pod-stop-time")

	// Setup synchronization between sliders and textboxes
	setupSliderSync(controls.sliderCPUUsage, nil)
	setupSliderSync(controls.sliderPODCPURequest, nil)
	setupSliderSync(controls.sliderPODCPULimit, nil)
	setupSliderSync(controls.sliderHPAMinReplicas, nil)
	setupSliderSync(controls.sliderHPAMaxReplicas, nil)
	setupSliderSync(controls.sliderHPATargetCPUUtilization, nil)
	setupSliderSync(controls.sliderNumberOfPods, nil)
	setupSliderSync(controls.sliderHistorySize, callbackHistorySize)
	setupSliderSync(controls.sliderScaleDownStabilizationWindow, nil)
	setupSliderSync(controls.sliderPODStartupTime, nil)
	setupSliderSync(controls.sliderPODStopTime, nil)

	return controls
}

func getSliderControl(document js.Value, sliderID, textboxID string) sliderControl {
	slider := document.Call("getElementById", sliderID)
	textBox := document.Call("getElementById", textboxID)
	return sliderControl{slider: slider, textBox: textBox}
}

func setupSliderSync(control sliderControl, callback func(string)) {
	// Synchronize slider and text box
	control.slider.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := control.slider.Get("value").String()
		control.textBox.Set("value", value)
		if callback != nil {
			callback(value)
		}
		return nil
	}))

	control.textBox.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := control.textBox.Get("value").String()
		control.slider.Set("value", value)
		if callback != nil {
			callback(value)
		}
		return nil
	}))
}

func updateChart(c *chart, newPodValue, starting, stopping, newPodLoad, newUnmetLoad int) {

	last := len(c.pods.data) - 1

	// shift left
	for i := 0; i < last; i++ {
		c.pods.data[i] = c.pods.data[i+1]
		c.podsStarting.data[i] = c.podsStarting.data[i+1]
		c.podsStopping.data[i] = c.podsStopping.data[i+1]
		c.podsLoad.data[i] = c.podsLoad.data[i+1]
		c.unmetLoad.data[i] = c.unmetLoad.data[i+1]
	}

	// add new value at the end
	c.pods.data[last] = newPodValue
	c.podsStarting.data[last] = starting
	c.podsStopping.data[last] = stopping
	c.podsLoad.data[last] = newPodLoad
	c.unmetLoad.data[last] = newUnmetLoad
}

func (c *chart) resizeHistory(newSize int) {
	if newSize == len(c.pods.data) {
		// no change
		return
	}

	c.pods.data = resizeSliceInt(c.pods.data, newSize)
	c.podsStarting.data = resizeSliceInt(c.podsStarting.data, newSize)
	c.podsStopping.data = resizeSliceInt(c.podsStopping.data, newSize)
	c.podsLoad.data = resizeSliceInt(c.podsLoad.data, newSize)
	c.unmetLoad.data = resizeSliceInt(c.unmetLoad.data, newSize)
}

func resizeSliceInt(oldSlice []int, newSize int) []int {
	if newSize == len(oldSlice) {
		// no change
		return oldSlice
	}

	newSlice := make([]int, newSize)

	// copy existing data to new slice
	copySize := min(len(oldSlice), newSize)

	copy(newSlice[newSize-copySize:], oldSlice[len(oldSlice)-copySize:])

	return newSlice
}

func newChart(ctxPods, ctxPodsLoad, ctxUnmetLoad,
	legendPods, legendPodsLoad, legendsUnmetLoad js.Value,
	canvasWidth, canvasHeight, historySize int) chart {
	c := chart{
		pods:         subchart{ctx: ctxPods, legend: legendPods, data: make([]int, historySize)},
		podsStarting: subchart{ctx: ctxPods, data: make([]int, historySize)},
		podsStopping: subchart{ctx: ctxPods, data: make([]int, historySize)},
		podsLoad:     subchart{ctx: ctxPodsLoad, legend: legendPodsLoad, data: make([]int, historySize)},
		unmetLoad:    subchart{ctx: ctxUnmetLoad, legend: legendsUnmetLoad, data: make([]int, historySize)},
		canvasWidth:  canvasWidth,
		canvasHeight: canvasHeight,
	}

	// fill pods with 1 (only for replicas)
	for i := range historySize {
		c.pods.data[i] = 1
	}

	return c
}

func drawCharts(ctxReplicas, ctxPodLoad, ctxUnmetLoad js.Value, c chart) {
	const drawLabels = true
	const hideLabels = false

	clearChart(ctxReplicas, c)
	{
		lo, hi := findMinMax(append(c.pods.data, append(c.podsStarting.data, c.podsStopping.data...)...))
		drawOneChart(ctxReplicas, c.pods.legend, c, c.pods.data, "blue", drawLabels, 6, lo, hi)
		drawOneChart(ctxReplicas, js.Null(), c, c.podsStarting.data, "green", hideLabels, 4, lo, hi)
		drawOneChart(ctxReplicas, js.Null(), c, c.podsStopping.data, "red", hideLabels, 2, lo, hi)
	}

	clearChart(ctxPodLoad, c)
	{
		lo, hi := findMinMax(c.podsLoad.data)
		drawOneChart(ctxPodLoad, c.podsLoad.legend, c, c.podsLoad.data, "blue", drawLabels, 2, lo, hi)
	}

	clearChart(ctxUnmetLoad, c)
	{
		lo, hi := findMinMax(c.podsLoad.data)
		drawOneChart(ctxUnmetLoad, c.unmetLoad.legend, c, c.unmetLoad.data, "blue", drawLabels, 2, lo, hi)
	}
}

func findMinMax(data []int) (int, int) {
	maxPods := 0
	minPods := math.MaxInt // NaN
	for _, v := range data {
		if v > maxPods {
			maxPods = v
		}
		if v < minPods {
			minPods = v
		}
	}
	return minPods, maxPods
}

func clearChart(ctx js.Value, c chart) {
	ctx.Set("fillStyle", "white")
	ctx.Call("fillRect", 0, 0, c.canvasWidth, c.canvasHeight)
}

func drawOneChart(ctx, legend js.Value, c chart, data []int,
	color string, drawLabels bool,
	width,
	minPods, maxPods int) {

	// draw a border
	/*
		ctx.Set("strokeStyle", "black")
		ctx.Set("lineWidth", 2)
		ctx.Call("strokeRect", 0, 0, c.canvasWidth, c.canvasHeight)
	*/

	// find max pod value
	/*
		maxPods := 0
		minPods := math.MaxInt // NaN
		for _, v := range data {
			if v > maxPods {
				maxPods = v
			}
			if v < minPods {
				minPods = v
			}
		}
	*/

	// pod space x ranges from 0 to len(c.pods)
	// pod space y ranges from 0 to maxPods
	// canvas space x ranges from 0 to c.canvasWidth
	// canvas space y ranges from 0 to c.canvasHeight

	// draw line
	ctx.Set("strokeStyle", color)
	ctx.Set("lineWidth", width)
	ctx.Call("beginPath")

	/*
		maxPodsShifted := maxPods - chartVerticalMinimum
	*/
	// avoid division by zero
	if maxPods <= 0 {
		maxPods = 1
	}

	// now draw considering chartVerticalMinimum

	for i, v := range data {
		// map pod space to canvas space
		x := i * c.canvasWidth / len(data)
		//y := c.canvasHeight - ((v - chartVerticalMinimum) * c.canvasHeight / maxPodsShifted) // invert y axis
		y := c.canvasHeight - (v * c.canvasHeight / maxPods) // invert y axis

		if i == 0 {
			ctx.Call("moveTo", x, y)
		} else {
			ctx.Call("lineTo", x, y)
		}
	}
	ctx.Call("stroke")

	if drawLabels {
		// Draw a label for max replicas at top-left corner
		labelText := fmt.Sprintf("Max: %d", maxPods)
		ctx.Set("font", "16px Arial")
		ctx.Set("fillStyle", "black")
		ctx.Call("fillText", labelText, 10, 20)

		// Draw a label for latest replicas count at right size
		// But vertically aligned with the last point
		latestReplicas := data[len(data)-1]

		labelText = fmt.Sprintf("Cur: %d", latestReplicas)
		textMetrics := ctx.Call("measureText", labelText)
		textWidth := textMetrics.Get("width").Float()

		x := c.canvasWidth - int(textWidth) - 5
		y := c.canvasHeight - (latestReplicas * c.canvasHeight / maxPods)
		// Move y slight up to avoid overlapping with the line
		y -= 10
		// Adjust y to avoid drawing outside canvas
		if y < 20 {
			y = 20
		}
		if y > c.canvasHeight-10 {
			y = c.canvasHeight - 10
		}

		ctx.Call("fillText", labelText, x, y)

		// Draw label with min, max, current replicas into legend element
		if !legend.IsNull() {
			var minPodsStr string
			if minPods == math.MaxInt {
				minPodsStr = "N/A"
			} else {
				minPodsStr = fmt.Sprintf("%d", minPods)
			}
			legendHTML := fmt.Sprintf("Min:%s Max:%d Current:%d",
				minPodsStr, maxPods, latestReplicas)
			legend.Set("innerHTML", legendHTML)
		}
	}
}
