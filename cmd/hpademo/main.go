// Package main implements the tool.
package main

import (
	"fmt"
	"strconv"
	"syscall/js"
)

const version = "0.0.1"

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
	document := js.Global().Get("document")

	hpaDemoVersion := fmt.Sprintf("hpademo %s", version)

	// add version to element with id "version"
	versionElement := document.Call("getElementById", "version")
	versionElement.Set("innerHTML", hpaDemoVersion)

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
	sliderHPAMinReplicas          sliderControl
	sliderHPAMaxReplicas          sliderControl
	sliderHPATargetCPUUtilization sliderControl
	sliderNumberOfPods            sliderControl
}

type sliderControl struct {
	slider  js.Value
	textBox js.Value
}

func addHTMLControls(document js.Value) podControls {
	var controls podControls

	// Get references to existing HTML elements
	controls.sliderCPUUsage = getSliderControl(document, "slider-cpu-usage", "textbox-cpu-usage")
	controls.sliderPODCPURequest = getSliderControl(document, "slider-pod-cpu-request", "textbox-pod-cpu-request")
	controls.sliderPODCPULimit = getSliderControl(document, "slider-pod-cpu-limit", "textbox-pod-cpu-limit")
	controls.sliderHPAMinReplicas = getSliderControl(document, "slider-hpa-min-replicas", "textbox-hpa-min-replicas")
	controls.sliderHPAMaxReplicas = getSliderControl(document, "slider-hpa-max-replicas", "textbox-hpa-max-replicas")
	controls.sliderHPATargetCPUUtilization = getSliderControl(document, "slider-hpa-target-cpu", "textbox-hpa-target-cpu")
	controls.sliderNumberOfPods = getSliderControl(document, "slider-number-of-pods", "textbox-number-of-pods")

	// Setup synchronization between sliders and textboxes
	setupSliderSync(controls.sliderCPUUsage)
	setupSliderSync(controls.sliderPODCPURequest)
	setupSliderSync(controls.sliderPODCPULimit)
	setupSliderSync(controls.sliderHPAMinReplicas)
	setupSliderSync(controls.sliderHPAMaxReplicas)
	setupSliderSync(controls.sliderHPATargetCPUUtilization)
	setupSliderSync(controls.sliderNumberOfPods)

	return controls
}

func getSliderControl(document js.Value, sliderID, textboxID string) sliderControl {
	slider := document.Call("getElementById", sliderID)
	textBox := document.Call("getElementById", textboxID)
	return sliderControl{slider: slider, textBox: textBox}
}

func setupSliderSync(control sliderControl) {
	// Synchronize slider and text box
	control.slider.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := control.slider.Get("value").String()
		control.textBox.Set("value", value)
		return nil
	}))

	control.textBox.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := control.textBox.Get("value").String()
		control.slider.Set("value", value)
		return nil
	}))
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
		pods:         make([]int, historySize),
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
