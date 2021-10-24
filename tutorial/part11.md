# Vulcan, Go, and A Triangle, Part 11

In this part, I used the previous created semaphores and fences to coordinate rendering and presentation. I also implement resizing of the window.

This part follows along with [Drawing a triangle / Drawing / Rendering and presentation / Acquiring an image from the swap chain](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Rendering_and_presentation#page_Acquiring-an-image-from-the-swap-chain) through to the end of [Drawing a triangle / Swap chain recreation](https://vulkan-tutorial.com/Drawing_a_triangle/Swap_chain_recreation#page_Handling-minimization).

<!--more-->

## Graphics Queue

In the last part, I added the ability to execute multiple frames in parallel. Before I could do much else in drawFrame, I needed to move the counter forward each time a frame is drawn. I also added the skeleton for the other bits I was going to need:

```golang
func (app *TriangleApplication) drawFrame() {
	// Wait for Vulkan to finish with this frame.

	// Get the index of the next image.

	// Wait for Vulkan to finish with this image.

	// Update inflight fences.

	// Create the graphics queue submit info object.

	// Reset the fence for this frame.

	// Submit work to the graphics queue.

	// Create the present queue info object.

	// Submit work to the present queue.

	// Update the current frame.
	app.currentFrame = (app.currentFrame + 1) % app.FramesInFlight
}
```

In theory, I could use the same `vk.DeviceWaitIdle(app.device)` used in the recreate pipeline function to prevent concurrency issues, but that would be wasteful. Instead I start by adding the fence waits.

```golang
func (app *TriangleApplication) drawFrame() {
	// Wait for Vulkan to finish with this frame.
	vk.WaitForFences(app.device,
		1,
		app.inFlightFences[app.currentFrame:],
		vk.True,
		vk.MaxUint64)

	...
	// Reset the fence for this frame.
	vk.ResetFences(app.device,
		1,
		app.inFlightFences[app.currentFrame:])

	...
}
```

Because of how the Vulkan APIs work, I was able to reference a subslice through to the end, but specify a fence count of 1 to make sure I only waited on the current fence. The `vk.True` tells the API to wait until all fences -- just the one in our case -- have been singled. The `vk.MaxUint64` is the timeout to wait, in nanoseconds. The device may not actually support nanosecond level resolution, but the API dictates that is the unit of measure.

In theory, since nothing could have signaled this fence yet, we should block and never get started. In part 10, when we created the fences, we set them all to be [created in a signaled state](https://github.com/ibd1279/vulkangotriangle/blob/54fe287531939e9b15b70b1c99e30bfadf2efee4/main.go#L309).

Later in the function -- just before submitting work to the graphics queue, I reset the fence to prevent this frame number from being processed until Vulkan has finished the submitted work.

The next step was to get the image from the swapchain. 

```golang
func (app *TriangleApplication) drawFrame() {
	...
	// Get the index of the next image.
	var imageIndex uint32
	ret := vk.AcquireNextImage(app.device,
		app.pipeline.Swapchain,
		vk.MaxUint64,
		app.imageAvailableSemaphores[app.currentFrame],
		vk.Fence(vk.NullHandle),
		&imageIndex)
	if ret == vk.ErrorOutOfDate {
		app.recreatePipeline()
		return
	} else if ret != vk.Success && ret != vk.Suboptimal {
		panic(vk.Error(ret))
	}
	
	...
}
```

The `vk.MaxUint64` in the AcquireNextImage call is a timeout, similar to WaitForFences. One could pass a value of zero for the timeout and make the call async. In order to safely use the value, it would be necessary to pass a non-null handle to the `vk.Fence(vk.NullHandle)` parameter. The `app.imageAvailableSemaphores` parameter works the same way.

The result code is handled differently here. If the result from acquiring the image was that the pipeline is out of date, I recreate the pipeline and let the frame try again. If the result of getting a frame was suboptimal or success, we continue with this frame, and will deal with it at the end of the frame.

With the imageIndex, I could check to see if this image was already in flight.

```golang
func (app *TriangleApplication) drawFrame() {
	...
	// Wait for Vulkan to finish with this image.
	if app.imagesInFlight[imageIndex] != vk.Fence(vk.NullHandle) {
		vk.WaitForFences(app.device,
			1,
			app.imagesInFlight[imageIndex:],
			vk.True,
			vk.MaxUint64)
	}

	// Update inflight fences.
	app.imagesInFlight[imageIndex] = app.inFlightFences[app.currentFrame]

	...
}
```

This is similar to the previous `WaitForFences()` for the current frame. The difference is that it uses the fence associated with the image index, rather than the current frame. This addresses the issue where an image index is acquired out of order. Once the fence is signaled, I updated the fence associated with the image index.

Next I submit the work to the graphics queue. This happens around the call to reset the fence.

```golang
func (app *TriangleApplication) drawFrame() {
	...
	// Create the graphics queue submit info object.
	submitInfos := []vk.SubmitInfo{
		vk.SubmitInfo{
			SType:              vk.StructureTypeSubmitInfo,
			WaitSemaphoreCount: 1,
			PWaitSemaphores: []vk.Semaphore{
				app.imageAvailableSemaphores[app.currentFrame],
			},
			PWaitDstStageMask: []vk.PipelineStageFlags{
				vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
			},
			CommandBufferCount: 1,
			PCommandBuffers: []vk.CommandBuffer{
				app.pipeline.GraphicsCommandBuffers[imageIndex],
			},
			SignalSemaphoreCount: 1,
			PSignalSemaphores: []vk.Semaphore{
				app.renderFinishedSemaphores[app.currentFrame],
			},
		},
	}

	...

	// Submit work to the graphics queue.
	MustSucceed(vk.QueueSubmit(app.graphicsQueue, 1, submitInfos, app.inFlightFences[app.currentFrame]))

	...
}
```

The Vulkan Tutorial has a better explanation than I do on the [purpose of these fields](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Rendering_and_presentation#page_Submitting-the-command-buffer). At a high level, these fields configured the values to wait on (images being available and pipeline stages), what work to do (the command buffers), and what to signal when finished (semaphores). This was submitted to the graphics queue, with the current frames fence for signaling when the work is finished.

## Presentation Queue

When I submitted work to the graphics queue, I provided semaphores to coordinate with the presentation queue. That allows me to submit the work to the presentation queue, using those same semaphores, so that the presentation queue doesn't start presenting until the graphics queue has signed it is done.

```golang
func (app *TriangleApplication) drawFrame() {
	...
	// Create the present queue info object.
	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores: []vk.Semaphore{
			app.renderFinishedSemaphores[app.currentFrame],
		},
		SwapchainCount: 1,
		PSwapchains: []vk.Swapchain{
			app.pipeline.Swapchain,
		},
		PImageIndices: []uint32{imageIndex},
	}

	// Submit work to the present queue.
	ret = vk.QueuePresent(app.presentationQueue, &presentInfo)
	if ret == vk.ErrorOutOfDate || ret == vk.Suboptimal {
		app.recreatePipeline()
	} else if ret != vk.Success {
		panic(fmt.Errorf("Failed to acquire next image. result %d.", ret))
	}

	...
}
```

## Cleanup

I added lots of synchronization to the drawing routine, but I neglected to add synchronization to the cleanup. The easiest method was to add `vk.DeviceWaitIdle(device)` to the beginning of the pipeline cleanup method.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	vk.DeviceWaitIdle(device)
	...
}
```

## Resizing

I had already added the primary logic for recreating the pipeline into the `drawFrame()` function. But the Vulkan tutorial goes further and I wanted to copy their completeness.

I started by adding a field to track GLFW resizes.

```golang
type TriangleApplication struct {
	...
	
	framebufferResize bool
}
```

Then I updated the create window function to allow resizing and provided a callback on resize.

```golang
func (app *TriangleApplication) setup() {
	// Steps.
	createWindow := func() {
		// Initialize GLFW
		err := glfw.Init()
		if err != nil {
			panic(err)
		}

		// Tell GLFW we aren't using OpenGL.
		glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)

		// Create the window object.
		app.window, err = glfw.CreateWindow(WindowWidth, WindowHeight, "Vulkan", nil, nil)
		if err != nil {
			panic(err)
		}

		// Callback for the framebuffer size changing.
		app.window.SetFramebufferSizeCallback(func(*glfw.Window, int, int) {
			app.framebufferResize = true
		})

		// Update required extensions.
		app.RequiredInstanceExtensionNames = append(
			app.RequiredInstanceExtensionNames,
			app.window.GetRequiredInstanceExtensions()...,
		)
	}
	...
}
```

> **Note:** It isn't as obvious, but I removed `glfw.WindowHint(glfw.Resizable, glfw.False)` from the function.

Then, I updated the if statement at the end of drawFrame() to also check if the callback flag had been resized.

```golang
func (app *TriangleApplication) drawFrame() {
	...
	// Submit work to the present queue.
	ret = vk.QueuePresent(app.presentationQueue, &presentInfo)
	if ret == vk.ErrorOutOfDate || ret == vk.Suboptimal || app.framebufferResize {
		app.recreatePipeline()
	} else if ret != vk.Success {
		panic(fmt.Errorf("Failed to acquire next image. result %d.", ret))
	}
	...
}
```

Finally, I needed to deal with minimization, or a scenario where the surface size from GLFW is zero. I reset the value of the resize flag after that.

```golang
func (app *TriangleApplication) recreatePipeline() {
	// wait if the current framebuffer surface is 0
	width, height := app.window.GetFramebufferSize()
	for width == 0 || height == 0 {
		width, height = app.window.GetFramebufferSize()
		glfw.WaitEvents()
	}

	// Clear the framebuffer resize flag.
	app.framebufferResize = false
	...
}
```

And now you can resize the colorful triangle window.

## Conclusion

This part concluded the [Drawing a triangle](https://vulkan-tutorial.com/Drawing_a_triangle/Swap_chain_recreation#page_Handling-minimization) chapter of the Vulkan Tutorial. I have a well behaved Vulkan application with a working pipeline.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/842525e9bc16f424b9251a8ad3397882a8fb07c9/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/842525e9bc16f424b9251a8ad3397882a8fb07c9#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261)) / [pipeline.go](https://github.com/ibd1279/vulkangotutorial/blob/842525e9bc16f424b9251a8ad3397882a8fb07c9/pipeline.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/842525e9bc16f424b9251a8ad3397882a8fb07c9#diff-39628d7d0647cdeebd6490044d9c16b508976806719f989b21c058ec71258fe0))
