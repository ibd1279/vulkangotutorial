# Vulcan, Go, and A Triangle, Part 8

In this part, I made the image views, render pass, framebuffers and pipeline layout. All things that only modify the pipeline file. I found [Drawing a triangle / Graphics pipeline basics / Introduction](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Introduction) an excellent reminder about computer graphics in general and useful for understanding vulkan in particular. 

This part doesn't relate to a single section in the Vulkan tutorial; it jumps around between a couple of different sections that were all pipeline specific and would need to be recreated if the pipeline needed to be recreated. It also references the Vulkan Tutorial almost constantly, as I didn't want to plagiarize their excellent explanations of these concepts.

<!--more-->

## Image views

Starting with [Drawing a triangle / Presentation / Image views](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Image_views), I updated the pipeline with a field for image views.

```golang
type Pipeline struct {
	Swapchain            vk.Swapchain
	SwapchainImages      []vk.Image
	SwapchainImageFormat vk.Format
	SwapchainExtent      vk.Extent2D
	SwapchainImageViews  []vk.ImageView
}
```

Then I added the image views skeleton into the new pipeline function.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	
	// Create the image views.
	imageViews := func() []vk.ImageView {}()
	
	// Create and return the pipeline
	return &Pipeline{
		Swapchain:            swapchain,
		SwapchainImages:      swapchainImages,
		SwapchainImageFormat: format.Format,
		SwapchainExtent:      extent,
		SwapchainImageViews:  imageViews,
	}
}
```

I needed to create one image view per swapchain image. This means that I had to change up the create pattern order a little bit and moved the result object before the info object.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the image views.
	imageViews := func() []vk.ImageView {
		// Create the result object.
		imageViews := make([]vk.ImageView, len(swapchainImages))

		// Create one image view per image.
		for k, img := range swapchainImages {
			// Create the info object.
			
			// Call the Vulkan function.
		}
		
		// return the image views
		return imageViews
	}()
	...
}
```

The body of the for loop is simply creating an info object and calling the vulkan function to create them.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	imageViews := func() []vk.ImageView {
		...
		for k, img := range swapchainImages {
			// Create the info object.
			imageViewInfo := vk.ImageViewCreateInfo{
				SType:    vk.StructureTypeImageViewCreateInfo,
				Image:    img,
				ViewType: vk.ImageViewType2d,
				Format:   format.Format,
				Components: vk.ComponentMapping{
					R: vk.ComponentSwizzleIdentity,
					G: vk.ComponentSwizzleIdentity,
					B: vk.ComponentSwizzleIdentity,
					A: vk.ComponentSwizzleIdentity,
				},
				SubresourceRange: vk.ImageSubresourceRange{
					AspectMask:     vk.ImageAspectFlags(vk.ImageAspectColorBit),
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			}

			// Call the Vulkan function.
			MustSucceed(vk.CreateImageView(app.device, &imageViewInfo, nil, &imageViews[k]))
		}
		...
	}()
	...
}
```

More details on each of these fields is available in the [Vulkan Tutorial on Image Views](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Image_views).

Finally, I needed to clean up the image views, which also requires a loop.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	for _, imgView := range pipeline.SwapchainImageViews {
		vk.DestroyImageView(device, imgView, nil)
	}
	vk.DestroySwapchain(device, pipeline.Swapchain, nil)
}
```

## Render pass

I jumped over to [Drawing a triangle / Graphics pipeline basics / Render passes](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Render_passes#page_Render-pass). The render pass is necessary to create the framebuffers, and I considered the framebuffers the final part of creating the swapchain.

I added the RenderPass field at the bottom of the pipeline type.

```golang
type Pipeline struct {
	...

	RenderPass vk.RenderPass
}
```

Then I added the skeleton to the New pipeline function.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...

	// Create the render pass.
	renderPass := func() vk.RenderPass {}()

	// Create and return the pipeline
	return &Pipeline{
		Swapchain:            swapchain,
		SwapchainImages:      swapchainImages,
		SwapchainImageFormat: format.Format,
		SwapchainExtent:      extent,
		SwapchainImageViews:  imageViews,
		RenderPass:           renderPass,
	}
}
```

The function was filled in with a create pattern. The constant values come directly from the [Vulkan Tutorial on Render passes](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Render_passes).

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the render pass.
	renderPass := func() vk.RenderPass {
		// Create the info object.
		renderPassInfo := vk.RenderPassCreateInfo{
			SType:           vk.StructureTypeRenderPassCreateInfo,
			AttachmentCount: 1,
			PAttachments: []vk.AttachmentDescription{
				vk.AttachmentDescription{
					Format:         format.Format,
					Samples:        vk.SampleCount1Bit,
					LoadOp:         vk.AttachmentLoadOpClear,
					StoreOp:        vk.AttachmentStoreOpStore,
					StencilLoadOp:  vk.AttachmentLoadOpDontCare,
					StencilStoreOp: vk.AttachmentStoreOpDontCare,
					InitialLayout:  vk.ImageLayoutUndefined,
					FinalLayout:    vk.ImageLayoutPresentSrc,
				},
			},
			SubpassCount: 1,
			PSubpasses: []vk.SubpassDescription{
				vk.SubpassDescription{
					PipelineBindPoint:    vk.PipelineBindPointGraphics,
					ColorAttachmentCount: 1,
					PColorAttachments: []vk.AttachmentReference{
						vk.AttachmentReference{
							Attachment: 0,
							Layout:     vk.ImageLayoutColorAttachmentOptimal,
						},
					},
				},
			},
			DependencyCount: 1,
			PDependencies: []vk.SubpassDependency{
				vk.SubpassDependency{
					SrcSubpass:    vk.SubpassExternal,
					SrcStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
					DstStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
					DstAccessMask: vk.AccessFlags(vk.AccessColorAttachmentWriteBit),
				},
			},
		}

		// Create the result object.
		var renderPass vk.RenderPass

		// Call the Vulkan function.
		MustSucceed(vk.CreateRenderPass(app.device, &renderPassInfo, nil, &renderPass))

		// return the render pass
		return renderPass
	}()
	...
}
```

The values I used for the `PDependencies` field are actually from [Subpass dependencies](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Rendering_and_presentation#page_Subpass-dependencies) in the rendering section of the Vulkan Tutorial. The values were constants, so it seemed worth pulling them into this section to avoid a more complicated diff later.

I couldn't forget the cleanup.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	vk.DestroyRenderPass(device, pipeline.RenderPass, nil)
	...
}
```

I was ready to create the framebuffers.

## Framebuffers

The [Vulkan tutorial does Framebuffers](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Framebuffers) much later in the process, as they are associated with drawing. I'm doing them earlier in the process because I associate them with the swapchain.

Exactly like the image views, I created one framebuffer per image view. I started by adding the field to the pipeline type.

```golang
type Pipeline struct {
	Swapchain             vk.Swapchain
	SwapchainImages       []vk.Image
	SwapchainImageFormat  vk.Format
	SwapchainExtent       vk.Extent2D
	SwapchainImageViews   []vk.ImageView
	SwapchainFramebuffers []vk.Framebuffer

	...
}
```

I Added the skeleton to the New pipeline function.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the framebuffers.
	framebuffers := func() []vk.Framebuffer {}()

	// Create and return the pipeline
	return &Pipeline{
		Swapchain:             swapchain,
		SwapchainImages:       swapchainImages,
		SwapchainImageFormat:  format.Format,
		SwapchainExtent:       extent,
		SwapchainImageViews:   imageViews,
		SwapchainFramebuffers: framebuffers,
		RenderPass:            renderPass,
	}
}
```

And I populated the body of the framebuffers function. This looks very similar to the image views looped creation.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the framebuffers.
	framebuffers := func() []vk.Framebuffer {
		// Create the result object.
		buffers := make([]vk.Framebuffer, len(imageViews))

		// Create one framebuffer per image view.
		for k, imgView := range imageViews {
			// Create the info object.
			bufferInfo := vk.FramebufferCreateInfo{
				SType:           vk.StructureTypeFramebufferCreateInfo,
				RenderPass:      renderPass,
				AttachmentCount: 1,
				PAttachments: []vk.ImageView{
					imgView,
				},
				Width:  extent.Width,
				Height: extent.Height,
				Layers: 1,
			}

			// Call the Vulkan function.
			MustSucceed(vk.CreateFramebuffer(app.device, &bufferInfo, nil, &buffers[k]))
		}
		
		// Return the framebuffers.
		return buffers
	}()
	...
}
```

The create info includes the render pass that will be using the frame buffer and the image view to attach to the frame buffer. The extent from the swapchain images is also used to describe the resolution. More details are available on the [Vulkan Tutorial on framebuffers](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Framebuffers).

Then I added the cleanup of the frame buffers.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	for _, buffer := range pipeline.SwapchainFramebuffers {
		vk.DestroyFramebuffer(device, buffer, nil)
	}

	vk.DestroyRenderPass(device, pipeline.RenderPass, nil)
	for _, imgView := range pipeline.SwapchainImageViews {
		vk.DestroyImageView(device, imgView, nil)
	}
	vk.DestroySwapchain(device, pipeline.Swapchain, nil)
}
```

With that, I was basically done with the swapchain related concepts. And ready to start building the pipeline related concepts onto of them.

## Pipeline layout

The pipeline layout is a creation pattern, and I won't be using much of the info struct fields at this point.

The first step was adding the PipelineLayout field to the structure.

```golang
type Pipeline struct {
	...
	RenderPass     vk.RenderPass
	PipelineLayout vk.PipelineLayout
}
```

Then I added the skeleton.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the pipeline layout.
	pipelineLayout := func() vk.PipelineLayout {}()

	// Create and return the pipeline
	return &Pipeline{
		Swapchain:             swapchain,
		SwapchainImages:       swapchainImages,
		SwapchainImageFormat:  format.Format,
		SwapchainExtent:       extent,
		SwapchainImageViews:   imageViews,
		SwapchainFramebuffers: framebuffers,
		RenderPass:            renderPass,
		PipelineLayout:        pipelineLayout,
	}
}
```

I populated the body with a create pattern.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the pipeline layout.
	pipelineLayout := func() vk.PipelineLayout {
		// Create the info object.
		layoutInfo := vk.PipelineLayoutCreateInfo{
			SType: vk.StructureTypePipelineLayoutCreateInfo,
		}

		// Create the result object.
		var layout vk.PipelineLayout

		// Call the Vulkan function.
		MustSucceed(vk.CreatePipelineLayout(app.device, &layoutInfo, nil, &layout))

		// Return the layout.
		return layout
	}()
	...
}
```

And finally I cleaned it up.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	for _, buffer := range pipeline.SwapchainFramebuffers {
		vk.DestroyFramebuffer(device, buffer, nil)
	}

	vk.DestroyPipelineLayout(device, pipeline.PipelineLayout, nil)
	vk.DestroyRenderPass(device, pipeline.RenderPass, nil)
	...
}
```

I'm mostly keeping frame buffers at the top of the file to match the order called in the Vulkan Tutorial.

## Conclusion

I added a lot of code. I created a lot of objects. The output still looks the same. That is because Each of these parts: the image views, the frame buffers, the render pass, and the pipeline layout, are all pieces of the pipeline. Vulkan is low level API and requires the application to be explicit about what it wants. As a result, it requires creating each of these in turn to configure the pipeline.
	
I am now up to [Drawing a triangle / Graphics pipeline basics / Introduction](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Introduction), although I've skipped a head and completed a couple of other sections in advance. In the next part, I'll load the shader modules and create vulkan pipeline object.

[pipeline.go](https://github.com/ibd1279/vulkangotutorial/blob/35ed1d96ac36c19ebb7cc0bb4493eec48c69da90/pipeline.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/35ed1d96ac36c19ebb7cc0bb4493eec48c69da90#diff-39628d7d0647cdeebd6490044d9c16b508976806719f989b21c058ec71258fe0))

