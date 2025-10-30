// Package main implements the tool.
package main

import (
	"fmt"
	"math"
	"strconv"
	"syscall/js"
)

type subchart struct {
	ctx    js.Value
	legend js.Value
	data   []int
}

type chart struct {
	pods         subchart
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

	// get canvas width and height
	canvasWidth := canvasPods.Get("width").Int()
	canvasHeight := canvasPods.Get("height").Int()

	const historySize = 300

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
			newPodValue = runHPADemoSimulation(controls)
			// update number of pods slider to reflect HPA decision
			controls.sliderNumberOfPods.slider.Set("value", newPodValue)
			controls.sliderNumberOfPods.textBox.Set("value", newPodValue)
		} else {
			// get from slider
			newPodValue = getSliderValueAsInt(controls.sliderNumberOfPods.slider)
		}

		//
		// evaluate per pod load
		//

		currentPods := float64(getSliderValueAsInt(controls.sliderNumberOfPods.slider))
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
		updateChart(&c, newPodValue, int(newPodLoad), int(newUnmetLoad))

		// redraw chart
		drawCharts(canvasPodsCtx, canvasPodsLoadCtx, canvasUnmetLoadCtx, c)

		return nil
	}), 1000)

	// prevent main from exiting

	fmt.Println("waiting forever...")
	select {}
}

type podControls struct {
	sliderCPUUsage                sliderControl
	sliderPODCPURequest           sliderControl
	sliderPODCPULimit             sliderControl
	sliderHPAMinReplicas          sliderControl
	sliderHPAMaxReplicas          sliderControl
	sliderHPATargetCPUUtilization sliderControl
	sliderNumberOfPods            sliderControl
	sliderHistorySize             sliderControl
}

type sliderControl struct {
	slider  js.Value
	textBox js.Value
}

func addHTMLControls(document js.Value, callbackHistorySize func(string)) podControls {

	// retrieve the container "controls" from the HTML
	uiControls := document.Call("getElementById", "controls")

	// create an element for stacking slider controls vertically
	container := document.Call("createElement", "div")
	container.Get("style").Set("display", "flex")
	container.Get("style").Set("flexDirection", "column")
	container.Get("style").Set("gap", "10px")
	uiControls.Call("appendChild", container)

	var controls podControls

	// create a slider for Total CPU Usage
	controls.sliderCPUUsage = createSliderWithTextBoxAndLabel(document, container, "Total CPU Usage (mCores)", 10, 100000, 100, nil)

	// create a slider for POD CPU Request
	controls.sliderPODCPURequest = createSliderWithTextBoxAndLabel(document, container, "POD CPU Request (mCores)", 10, 10000, 200, nil)

	// create a slider for POD CPU Limit
	controls.sliderPODCPULimit = createSliderWithTextBoxAndLabel(document, container, "POD CPU Limit (mCores)", 10, 10000, 600, nil)

	const (
		minPods = 1
		maxPods = 1000
	)

	// create a slider for HPA Min Pods
	controls.sliderHPAMinReplicas = createSliderWithTextBoxAndLabel(document, container, "HPA Min Replicas", minPods, maxPods, 1, nil)

	// create a slider for HPA Max Pods
	controls.sliderHPAMaxReplicas = createSliderWithTextBoxAndLabel(document, container, "HPA Max Replicas", minPods, maxPods, 10, nil)

	// create a slider for HPA Target CPU Utilization
	controls.sliderHPATargetCPUUtilization = createSliderWithTextBoxAndLabel(document, container, "HPA Target CPU Utilization", 1, 200, 80, nil)

	// create a slider for Number of Pods
	controls.sliderNumberOfPods = createSliderWithTextBoxAndLabel(document, container, "Number of Pods", minPods, maxPods, 2, nil)

	// create a slider for history size in seconds
	controls.sliderHistorySize = createSliderWithTextBoxAndLabel(document, container, "History Size (seconds)", 60, 1800, 300, callbackHistorySize)

	return controls
}

func createSliderWithTextBoxAndLabel(document js.Value,
	container js.Value, labelText string, minValue, maxValue, initial int,
	callback func(string)) sliderControl {

	// create a child control container to hold label, slider and text box
	controlContainer := document.Call("createElement", "div")
	controlContainer.Set("className", "control-item") // ADIÇÃO: classe para estilização
	controlContainer.Get("style").Set("display", "flex")
	controlContainer.Get("style").Set("flexDirection", "column")
	controlContainer.Get("style").Set("gap", "5px")
	controlContainer.Get("style").Set("padding", "10px")
	controlContainer.Get("style").Set("borderRadius", "8px")
	controlContainer.Get("style").Set("backgroundColor", "#f8f9fa")
	controlContainer.Get("style").Set("transition", "all 0.3s") // ADIÇÃO: transição suave
	container.Call("appendChild", controlContainer)

	// create a label (MOVIDO PARA CIMA)
	label := document.Call("createElement", "label")
	label.Set("innerHTML", labelText)
	label.Get("style").Set("fontSize", "12px")
	label.Get("style").Set("fontWeight", "600")
	label.Get("style").Set("color", "#374151")
	label.Get("style").Set("marginBottom", "4px")
	controlContainer.Call("appendChild", label)

	// create a container for slider and textbox in the same row
	inputRow := document.Call("createElement", "div")
	inputRow.Get("style").Set("display", "flex")
	inputRow.Get("style").Set("gap", "8px")
	inputRow.Get("style").Set("alignItems", "center")
	controlContainer.Call("appendChild", inputRow)

	// create a slider
	slider := document.Call("createElement", "input")
	slider.Set("type", "range")
	slider.Set("min", minValue)
	slider.Set("max", maxValue)
	slider.Set("value", initial)
	slider.Get("style").Set("flex", "1")
	slider.Get("style").Set("cursor", "pointer")
	inputRow.Call("appendChild", slider)

	// create a text box
	textBox := document.Call("createElement", "input")
	textBox.Set("type", "number")
	textBox.Set("min", minValue)
	textBox.Set("max", maxValue)
	textBox.Set("value", initial)
	textBox.Get("style").Set("width", "70px")
	textBox.Get("style").Set("padding", "4px 8px")
	textBox.Get("style").Set("border", "1px solid #d1d5db")
	textBox.Get("style").Set("borderRadius", "4px")
	textBox.Get("style").Set("fontSize", "14px")
	textBox.Get("style").Set("backgroundColor", "white") // ADIÇÃO: fundo branco
	inputRow.Call("appendChild", textBox)

	// synchronize slider and text box
	slider.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := slider.Get("value").String()
		textBox.Set("value", value)
		if callback != nil {
			callback(value)
		}
		return nil
	}))

	textBox.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := textBox.Get("value").String()
		slider.Set("value", value)
		if callback != nil {
			callback(value)
		}
		return nil
	}))

	return sliderControl{slider: slider, textBox: textBox}
}

func updateChart(c *chart, newPodValue, newPodLoad, newUnmetLoad int) {

	last := len(c.pods.data) - 1

	// shift left
	for i := 0; i < last; i++ {
		c.pods.data[i] = c.pods.data[i+1]
		c.podsLoad.data[i] = c.podsLoad.data[i+1]
		c.unmetLoad.data[i] = c.unmetLoad.data[i+1]
	}

	// add new value at the end
	c.pods.data[last] = newPodValue
	c.podsLoad.data[last] = newPodLoad
	c.unmetLoad.data[last] = newUnmetLoad
}

func (c *chart) resizeHistory(newSize int) {
	if newSize == len(c.pods.data) {
		// no change
		return
	}

	c.pods.data = resizeSliceInt(c.pods.data, newSize)
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
		podsLoad:     subchart{ctx: ctxPodsLoad, legend: legendPodsLoad, data: make([]int, historySize)},
		unmetLoad:    subchart{ctx: ctxUnmetLoad, legend: legendsUnmetLoad, data: make([]int, historySize)},
		canvasWidth:  canvasWidth,
		canvasHeight: canvasHeight,
	}

	// fill pods with 1 (only for replicas)
	for i := 0; i < historySize; i++ {
		c.pods.data[i] = 1
	}

	/*
		// fill pods
		for i := 0; i < historySize; i++ {
			// for every point, shift left and add a new value

			// loop to shift left
			for j := 0; j < len(c.pods)-1; j++ {
				c.pods[j] = c.pods[j+1]
			}

			// push a increasing value
			c.pods[len(c.pods)-1] = i
		}
	*/

	return c
}

func drawCharts(ctxReplicas, ctxPodLoad, ctxUnmetLoad js.Value, c chart) {
	drawOneChartLine(ctxReplicas, c.pods.legend, c, c.pods.data, 1)
	drawOneChartLine(ctxPodLoad, c.podsLoad.legend, c, c.podsLoad.data, 0)
	drawOneChartLine(ctxUnmetLoad, c.unmetLoad.legend, c, c.unmetLoad.data, 0)
}

func drawOneChartLine(ctx, legend js.Value, c chart, data []int, chartVerticalMinimum int) {
	// clear canvas
	ctx.Set("fillStyle", "white")
	ctx.Call("fillRect", 0, 0, c.canvasWidth, c.canvasHeight)

	// draw a border
	/*
		ctx.Set("strokeStyle", "black")
		ctx.Set("lineWidth", 2)
		ctx.Call("strokeRect", 0, 0, c.canvasWidth, c.canvasHeight)
	*/

	// find max pod value
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

	// pod space x ranges from 0 to len(c.pods)
	// pod space y ranges from 0 to maxPods
	// canvas space x ranges from 0 to c.canvasWidth
	// canvas space y ranges from 0 to c.canvasHeight

	// draw line
	ctx.Set("strokeStyle", "blue")
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")

	maxPodsShifted := maxPods - chartVerticalMinimum
	// avoid division by zero
	if maxPodsShifted <= 0 {
		maxPodsShifted = 1
	}

	// now draw considering chartVerticalMinimum

	for i, v := range data {
		// map pod space to canvas space
		x := i * c.canvasWidth / len(data)
		y := c.canvasHeight - ((v - chartVerticalMinimum) * c.canvasHeight / maxPodsShifted) // invert y axis

		if i == 0 {
			ctx.Call("moveTo", x, y)
		} else {
			ctx.Call("lineTo", x, y)
		}
	}
	ctx.Call("stroke")

	// Draw a label for max replicas at top-left corner
	labelText := fmt.Sprintf("Max: %d", maxPods)
	ctx.Set("font", "16px Arial")
	ctx.Set("fillStyle", "black")
	ctx.Call("fillText", labelText, 10, 20)

	// Draw a label for latest replicas count at right size
	// But vertically aligned with the last point
	latestReplicas := data[len(data)-1]
	x := c.canvasWidth - 60
	y := c.canvasHeight - ((latestReplicas - chartVerticalMinimum) * c.canvasHeight / maxPodsShifted)
	// Move y slight up to avoid overlapping with the line
	y -= 10
	// Adjust y to avoid drawing outside canvas
	if y < 20 {
		y = 20
	}
	if y > c.canvasHeight-10 {
		y = c.canvasHeight - 10
	}
	labelText = fmt.Sprintf("Cur: %d", latestReplicas)
	ctx.Call("fillText", labelText, x, y)

	// Draw label with min, max, current replicas into legend element
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
