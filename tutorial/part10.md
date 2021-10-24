# Vulcan, Go, and A Triangle, Part 10

In this part, I created the command pool, the command buffers, and recorded the rendering to our framebuffers.

This part is a direct translation of [Drawing a triangle / Drawing / Command buffers](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Command_buffers).

<!--more-->

## Command Pool

The command pool has a longer lifespan than the swapchain and pipelines, so I managed its lifecycle as part of the main application.

I started by adding the command pool field to the Triangle application in `main.go`.

```golang
type TriangleApplication struct {
	...
	
	pipeline            *Pipeline
	graphicsCommandPool vk.CommandPool
}
```

Then I populated the previous created `createCommandPool` function with the usual create pattern:

```golang
func (app *TriangleApplication) setup() {
	...
	createCommandPool := func() {
		// Get the queue families
		gIdx, _ := app.physicalDevice.QueueFamilies(app.surface)

		// Create the info object.
		poolInfo := vk.CommandPoolCreateInfo{
			SType:            vk.StructureTypeCommandPoolCreateInfo,
			QueueFamilyIndex: gIdx.Val(),
		}

		// Create the result object.
		var commandPool vk.CommandPool

		// Call the Vulkan function.
		MustSucceed(vk.CreateCommandPool(app.device, &poolInfo, nil, &commandPool))

		// Update the application.
		app.graphicsCommandPool = commandPool
	}
	
	...
}
```

Since the commands must be executed by one of the queues, I was required tie them to the queue family that will be executing the work. I'm only submitting work to the graphics queue family.

There are a couple of [flags](https://pkg.go.dev/github.com/vulkan-go/vulkan#CommandPoolCreateFlagBits) I could have specified on the create info, but none of them are required for now. The Vulkan Tutorial describes the [first two flags](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Command_buffers#page_Command-pools).

I needed to clean up the pool.

```golang
func (app *TriangleApplication) cleanup() {
	if app.pipeline != nil {
		app.pipeline.Cleanup(app.device)
	}

	vk.DestroyCommandPool(app.device, app.graphicsCommandPool, nil)
	...
}
```

## Command buffers

Now that I had a pool, I could start allocating and recording my command buffers. I'll be creating one command buffer per framebuffer. The command buffers will need references to the pipeline and the render pass and the vulkan pipeline, which impacts its location in the NewPipeline function.

I added the CommandBuffers field to the Pipeline type in the `pipeline.go` file because to clean them up later.

```golang
type Pipeline struct {
	...

	graphicsCommandPool    vk.CommandPool
	GraphicsCommandBuffers []vk.CommandBuffer
}
```

Then I added the skeleton to the NewPipeline function.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the command buffers.
	commandBuffers := func() []vk.CommandBuffer {
		// Create the info object.

		// Create the result object.

		// Call the vulkan function.

		// Record the commands.

		// Return the command buffers.
	}()

	// Create and return the pipeline
	return &Pipeline{
		Swapchain:              swapchain,
		SwapchainImages:        swapchainImages,
		SwapchainImageFormat:   format.Format,
		SwapchainExtent:        extent,
		SwapchainImageViews:    imageViews,
		SwapchainFramebuffers:  framebuffers,
		RenderPass:             renderPass,
		PipelineLayout:         pipelineLayout,
		Pipelines:              pipelines,
		graphicsCommandPool:    app.graphicsCommandPool,
		GraphicsCommandBuffers: commandBuffers,
	}
}
```

I captured a reference to the command pool here because I needed it later.

Allocating the command buffers is the usual Vulkan create pattern.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	commandBuffers := func() []vk.CommandBuffer {
		// Create the info object.
		buffersInfo := vk.CommandBufferAllocateInfo{
			SType:              vk.StructureTypeCommandBufferAllocateInfo,
			CommandPool:        app.graphicsCommandPool,
			Level:              vk.CommandBufferLevelPrimary,
			CommandBufferCount: uint32(len(framebuffers)),
		}

		// Create the result object.
		buffers := make([]vk.CommandBuffer, buffersInfo.CommandBufferCount)

		// Call the vulkan function.
		MustSucceed(vk.AllocateCommandBuffers(app.device, &buffersInfo, buffers))

		// Record the commands.

		// Return the command buffers.
		return buffers
	}()
	...
}
```

I implemented the recording of command buffers as a direct translation of the Vulkan Tutorial steps from [Starting command buffer recording](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Command_buffers#page_Starting-command-buffer-recording) to [Finishing up](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Command_buffers#page_Finishing-up).

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	commandBuffers := func() []vk.CommandBuffer {
		...
		// Record the commands.
		for k, cmdBuffer := range buffers {
			// Start recording
			MustSucceed(vk.BeginCommandBuffer(cmdBuffer, &vk.CommandBufferBeginInfo{
				SType: vk.StructureTypeCommandBufferBeginInfo,
			}))

			// Create the info object.
			beginInfo := vk.RenderPassBeginInfo{
				SType:       vk.StructureTypeRenderPassBeginInfo,
				RenderPass:  renderPass,
				Framebuffer: framebuffers[k],
				RenderArea: vk.Rect2D{
					Offset: vk.Offset2D{X: 0, Y: 0},
					Extent: extent,
				},
				ClearValueCount: 1,
				PClearValues: []vk.ClearValue{
					vk.NewClearValue([]float32{0.0, 0.0, 0.0, 1.0}),
				},
			}

			// Call the Vulkan function.
			vk.CmdBeginRenderPass(cmdBuffer, &beginInfo, vk.SubpassContentsInline)

			// Bind the buffer to the graphics point in the pipeline.
			vk.CmdBindPipeline(cmdBuffer, vk.PipelineBindPointGraphics, pipelines[0])

			// Draw
			vk.CmdDraw(cmdBuffer, 3, 1, 0, 0)

			// End the render pass
			vk.CmdEndRenderPass(cmdBuffer)

			// Stop recording
			MustSucceed(vk.EndCommandBuffer(cmdBuffer))
		}

		...
	}()
	...
}
```

I had tried doing the command recording in parallel using go-routines, but that resulted validation layer threading errors. A command pool is apparently thread specific.

The tutorial defers destroying the command buffers until the [recreation section](https://vulkan-tutorial.com/Drawing_a_triangle/Swap_chain_recreation#page_Recreating-the-swap-chain). Since I knew that was coming, I went ahead and added the cleanup to the Pipeline type.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	for _, buffer := range pipeline.SwapchainFramebuffers {
		vk.DestroyFramebuffer(device, buffer, nil)
	}

	vk.FreeCommandBuffers(device,
		pipeline.graphicsCommandPool,
		uint32(len(pipeline.GraphicsCommandBuffers)),
		pipeline.GraphicsCommandBuffers)

	for _, pl := range pipeline.Pipelines {
		vk.DestroyPipeline(device, pl, nil)
	}
	...
}
```

This is also why I captured the command pool reference earlier.

## Synchronization

In the next part, I will be rendering and presenting the results. In order for that to be successful, I need to address some parallel computing problems. The application (thread A) acquires an image from the swapchain, submits that image and a command buffer to an execution queue (thread B). Then the application (Thread A) returns the image to the presentation queue (thread B or C). This is complicated further by the fact that the problem isn't just multi-threading, but multiple cores with different physical access characteristics (GPU vs CPU).

Vulkan provides a couple of mechanisms to work around this: Semaphores and Fences. Semaphores are used to synchronize operations within Vulkan (In my case, between the graphics and presentation queues), while fences are designed to synchronize my application with Vulkan and the rendering process (In my case, not resubmitting an already inflight image).

I wanted the next part to be focused on the synchronization issues, so I pulled the object creation aspect into this part.

I added the fields for synchronization into the application in `main.go`.

```golang
type TriangleApplication struct {
	...

	imageAvailableSemaphores []vk.Semaphore
	renderFinishedSemaphores []vk.Semaphore
	inFlightFences           []vk.Fence
	imagesInFlight           []vk.Fence
	currentFrame             uint
	FramesInFlight           uint
}
```

Then, I provided the maximum number of frames in flight in the `main()` function.

```golang
func main() {
	app := TriangleApplication{
		...
		FramesInFlight: 2,
	}
	...
}
```

I populated the createSemaphores function inside `setup()`.

```golang
func (app *TriangleApplication) setup() {
	...
	createSemaphores := func() {
		// Create the info object.
		semaphoreInfo := vk.SemaphoreCreateInfo{
			SType: vk.StructureTypeSemaphoreCreateInfo,
		}

		// Create the result object(s).
		imgAvail := make([]vk.Semaphore, app.FramesInFlight)
		renderDone := make([]vk.Semaphore, app.FramesInFlight)

		// Call the Vulkan function...
		for h := 0; h < len(imgAvail); h++ {
			// ... for image available.
			MustSucceed(vk.CreateSemaphore(app.device, &semaphoreInfo, nil, &imgAvail[h]))

			// ... for render finished.
			MustSucceed(vk.CreateSemaphore(app.device, &semaphoreInfo, nil, &renderDone[h]))
		}

		// Update the application.
		app.imageAvailableSemaphores = imgAvail
		app.renderFinishedSemaphores = renderDone
	}
	
	...
}
```

And I did the same thing for the fences.

```golang
func (app *TriangleApplication) setup() {
	...
	createFences := func() {
		// Create the info object.
		fenceInfo := vk.FenceCreateInfo{
			SType: vk.StructureTypeFenceCreateInfo,
			Flags: vk.FenceCreateFlags(vk.FenceCreateSignaledBit),
		}

		// Create the result object.
		inFlightFences := make([]vk.Fence, app.FramesInFlight)

		// Call the Vulkan function.
		for k, _ := range inFlightFences {
			MustSucceed(vk.CreateFence(app.device, &fenceInfo, nil, &inFlightFences[k]))
		}

		// Update the application.
		app.inFlightFences = inFlightFences
	}
	
	...
}
```

And finally, I modified `recreatePipeline()` to wait for the device to be idle before recreating the pipeline and to resize the inflight images.

```golang
func (app *TriangleApplication) recreatePipeline() {
	// Wait for the device to finish work.
	vk.DeviceWaitIdle(app.device)

	...

	// Allocate Images in flight tracker.
	app.imagesInFlight = make([]vk.Fence, len(app.pipeline.SwapchainImages))
}
```

It is possible that a different pipeline could have a different number of swapchain images, so I resized the tracker based on the new pipeline swapchain.

Finally, I needed to clean up all these new objects.

```golang
func (app *TriangleApplication) cleanup() {
	if app.pipeline != nil {
		app.pipeline.Cleanup(app.device)
	}

	for _, fence := range app.inFlightFences {
		vk.DestroyFence(app.device, fence, nil)
	}
	for _, semaphore := range app.renderFinishedSemaphores {
		vk.DestroySemaphore(app.device, semaphore, nil)
	}
	for _, semaphore := range app.imageAvailableSemaphores {
		vk.DestroySemaphore(app.device, semaphore, nil)
	}
	...
}
```

Now I had all the pieces needed to coordinate the GPU and the CPU to render images.

## Conclusion

The pipeline is complete, including having the command buffers allocated and recorded. I've completed the Vulkan Tutorial up to [Drawing a triangle / Drawing / Rendering and presentation / Semaphores](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Rendering_and_presentation#page_Semaphores).

In the next part, I'll draw the triangle to that perpetually blank window, and enable resizing the window.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/009c7e7f9d01a2ee29f4e4a66baeffafd46ccfc4/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/009c7e7f9d01a2ee29f4e4a66baeffafd46ccfc4#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261)) / [pipeline.go](https://github.com/ibd1279/vulkangotutorial/commit/009c7e7f9d01a2ee29f4e4a66baeffafd46ccfc4#diff-39628d7d0647cdeebd6490044d9c16b508976806719f989b21c058ec71258fe0) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/009c7e7f9d01a2ee29f4e4a66baeffafd46ccfc4#diff-39628d7d0647cdeebd6490044d9c16b508976806719f989b21c058ec71258fe0))
