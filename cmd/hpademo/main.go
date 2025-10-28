// Package main implements the tool.
package main

import (
	"fmt"
	"syscall/js"
)

type chart struct {
	ctx          js.Value
	pods         []int
	canvasWidth  int
	canvasHeight int
}

func main() {
	fmt.Println("Hello, WebAssembly!!")

	// find canvas element
	document := js.Global().Get("document")
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

	// call updateChart every second
	js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// decrease pod value by 1
		newPodValue := c.pods[len(c.pods)-1] - 1
		if newPodValue < 0 {
			newPodValue = 0
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
