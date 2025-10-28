// Package main implements the tool.
package main

import (
	"fmt"
	"strconv"
	"syscall/js"
)

type chart struct {
	ctx          js.Value
	pods         []int
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
	fmt.Println("Hello, WebAssembly!!")

	document := js.Global().Get("document")

	controls := addHTMLControls(document)

	// find canvas element
	canvas := document.Call("getElementById", "canvas")

	// get canvas 2d context
	ctx := canvas.Call("getContext", "2d")

	// get canvas width and height
	canvasWidth := canvas.Get("width").Int()
	canvasHeight := canvas.Get("height").Int()

	const historySize = 300

	c := newChart(ctx, canvasWidth, canvasHeight, historySize)

	// call function to draw chart
	drawChart(ctx, c)

	var lastHPAEvaluation int

	// call updateChart every second
	js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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

		// update chart data
		updateChart(&c, newPodValue)

		// redraw chart
		drawChart(ctx, c)

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
	sliderHPAMinPods              sliderControl
	sliderHPAMaxPods              sliderControl
	sliderHPATargetCPUUtilization sliderControl
	sliderNumberOfPods            sliderControl
}

type sliderControl struct {
	slider  js.Value
	textBox js.Value
}

func addHTMLControls(document js.Value) podControls {

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
	controls.sliderCPUUsage = createSliderWithTextBoxAndLabel(document, container, "Total CPU Usage (mCores):", 10, 100000, 100)

	// create a slider for POD CPU Request
	controls.sliderPODCPURequest = createSliderWithTextBoxAndLabel(document, container, "POD CPU Request (mCores):", 10, 10000, 200)

	// create a slider for POD CPU Limit
	controls.sliderPODCPULimit = createSliderWithTextBoxAndLabel(document, container, "POD CPU Limit (mCores):", 10, 10000, 600)

	const (
		minPods = 1
		maxPods = 1000
	)

	// create a slider for HPA Min Pods
	controls.sliderHPAMinPods = createSliderWithTextBoxAndLabel(document, container, "HPA Min Pods:", minPods, maxPods, 1)

	// create a slider for HPA Max Pods
	controls.sliderHPAMaxPods = createSliderWithTextBoxAndLabel(document, container, "HPA Max Pods:", minPods, maxPods, 10)

	// create a slider for HPA Target CPU Utilization
	controls.sliderHPATargetCPUUtilization = createSliderWithTextBoxAndLabel(document, container, "HPA Target CPU Utilization:", 1, 200, 80)

	// create a slider for Number of Pods
	controls.sliderNumberOfPods = createSliderWithTextBoxAndLabel(document, container, "Number of Pods:", minPods, maxPods, 2)

	return controls
}

func createSliderWithTextBoxAndLabel(document js.Value,
	container js.Value, labelText string, minValue, maxValue, initial int) sliderControl {
	// create a label
	label := document.Call("createElement", "label")
	label.Set("innerHTML", labelText)
	container.Call("appendChild", label)

	// create a slider
	slider := document.Call("createElement", "input")
	slider.Set("type", "range")
	slider.Set("min", minValue)
	slider.Set("max", maxValue)
	slider.Set("value", initial)
	container.Call("appendChild", slider)

	// create a text box
	textBox := document.Call("createElement", "input")
	textBox.Set("type", "number")
	textBox.Set("min", minValue)
	textBox.Set("max", maxValue)
	textBox.Set("value", initial)
	container.Call("appendChild", textBox)

	// synchronize slider and text box
	slider.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := slider.Get("value").String()
		textBox.Set("value", value)
		return nil
	}))

	textBox.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := textBox.Get("value").String()
		slider.Set("value", value)
		return nil
	}))

	return sliderControl{slider: slider, textBox: textBox}
}

func updateChart(c *chart, newPodValue int) {
	// shift left
	for i := 0; i < len(c.pods)-1; i++ {
		c.pods[i] = c.pods[i+1]
	}

	// add new value at the end
	c.pods[len(c.pods)-1] = newPodValue
}

func newChart(ctx js.Value, canvasWidth, canvasHeight, historySize int) chart {
	c := chart{
		ctx:          ctx,
		pods:         make([]int, historySize, historySize),
		canvasWidth:  canvasWidth,
		canvasHeight: canvasHeight,
	}

	// fill with ones
	for i := 0; i < historySize; i++ {
		c.pods[i] = 1
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

func drawChart(ctx js.Value, c chart) {
	// clear canvas
	ctx.Set("fillStyle", "white")
	ctx.Call("fillRect", 0, 0, c.canvasWidth, c.canvasHeight)

	// draw a border
	ctx.Set("strokeStyle", "black")
	ctx.Set("lineWidth", 2)
	ctx.Call("strokeRect", 0, 0, c.canvasWidth, c.canvasHeight)

	// find max pod value
	maxPods := 0
	for _, v := range c.pods {
		if v > maxPods {
			maxPods = v
		}
	}

	lastPodValue := c.pods[len(c.pods)-1]
	fmt.Printf("maxPods=%d lastPodValue=%d historySize=%d\n", maxPods, lastPodValue, len(c.pods))

	// pod space x ranges from 0 to len(c.pods)
	// pod space y ranges from 0 to maxPods
	// canvas space x ranges from 0 to c.canvasWidth
	// canvas space y ranges from 0 to c.canvasHeight

	// draw line
	ctx.Set("strokeStyle", "blue")
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")

	for i, v := range c.pods {
		// map pod space to canvas space
		x := i * c.canvasWidth / len(c.pods)
		y := c.canvasHeight - (v * c.canvasHeight / maxPods) // invert y axis

		if i == 0 {
			ctx.Call("moveTo", x, y)
		} else {
			ctx.Call("lineTo", x, y)
		}
	}
	ctx.Call("stroke")
}
