package main

import (
	vk "github.com/vulkan-go/vulkan"
)

type Pipeline struct {
	Swapchain             vk.Swapchain
	SwapchainImages       []vk.Image
	SwapchainImageFormat  vk.Format
	SwapchainExtent       vk.Extent2D
	SwapchainImageViews   []vk.ImageView
	SwapchainFramebuffers []vk.Framebuffer

	RenderPass     vk.RenderPass
	PipelineLayout vk.PipelineLayout
	Pipelines      []vk.Pipeline

	graphicsCommandPool    vk.CommandPool
	GraphicsCommandBuffers []vk.CommandBuffer
}

func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		// Capture the old Swapchain.
		oldSwapchain := vk.Swapchain(vk.NullHandle)
		if oldPipeline != nil {
			oldSwapchain = oldPipeline.Swapchain
		}

		// Swapchain support.
		caps, fmts, modes := app.physicalDevice.SwapchainSupport(app.surface)

		// Formats.
		format := func() vk.SurfaceFormat {
			for _, v := range fmts {
				if v.Format == vk.FormatB8g8r8a8Srgb && v.ColorSpace == vk.ColorSpaceSrgbNonlinear {
					return v
				}
			}
			return fmts[0]
		}()

		// Present Mode.
		presentMode := func() vk.PresentMode {
			for _, v := range modes {
				if v == vk.PresentModeMailbox {
					return v
				}
			}
			return vk.PresentModeFifo
		}()

		// Extent.
		extent := func() vk.Extent2D {
			if caps.CurrentExtent.Width != vk.MaxUint32 {
				return caps.CurrentExtent
			} else {
				width, height := app.window.GetFramebufferSize()

				actualExtent := vk.Extent2D{
					Width:  uint32(width),
					Height: uint32(height),
				}

				actualExtent.Width = ClampUint32(actualExtent.Width,
					caps.MinImageExtent.Width,
					caps.MaxImageExtent.Width)
				actualExtent.Height = ClampUint32(actualExtent.Height,
					caps.MinImageExtent.Height,
					caps.MaxImageExtent.Height)

				return actualExtent
			}
		}()

		// Image Count.
		imgCount := func() uint32 {
			count := caps.MinImageCount + 1
			if caps.MaxImageCount > 0 {
				count = ClampUint32(count,
					caps.MinImageCount,
					caps.MaxImageCount)
			}
			return count
		}()

		// Queue Families and Share mode.
		qFamilyIndices, shareMode := func() ([]uint32, vk.SharingMode) {
			gIdx, pIdx := app.physicalDevice.QueueFamilies(app.surface)
			qfi := []uint32{gIdx.Val(), pIdx.Val()}
			sm := vk.SharingModeConcurrent
			if gIdx.Val() == pIdx.Val() {
				sm = vk.SharingModeExclusive
				qfi = qfi[:1]
			}
			return qfi, sm
		}()

		// Create the info object.
		swapchainInfo := vk.SwapchainCreateInfo{
			SType:                 vk.StructureTypeSwapchainCreateInfo,
			Surface:               app.surface,
			MinImageCount:         imgCount,
			ImageFormat:           format.Format,
			ImageColorSpace:       format.ColorSpace,
			ImageExtent:           extent,
			ImageArrayLayers:      1,
			ImageUsage:            vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit),
			ImageSharingMode:      shareMode,
			QueueFamilyIndexCount: uint32(len(qFamilyIndices)),
			PQueueFamilyIndices:   qFamilyIndices,
			PreTransform:          caps.CurrentTransform,
			CompositeAlpha:        vk.CompositeAlphaOpaqueBit,
			PresentMode:           presentMode,
			Clipped:               vk.True,
			OldSwapchain:          oldSwapchain,
		}

		// Create the result object.
		var swapchain vk.Swapchain

		// Call the Vulkan function.
		MustSucceed(vk.CreateSwapchain(app.device, &swapchainInfo, nil, &swapchain))

		// Fetch the Swapchain Images.
		var count uint32
		vk.GetSwapchainImages(app.device, swapchain, &count, nil)
		images := make([]vk.Image, count)
		vk.GetSwapchainImages(app.device, swapchain, &count, images)

		// return the swapchain and images.
		return swapchain, images, format, extent

	}()

	// Create the image views.
	imageViews := func() []vk.ImageView {
		// Create the result object.
		imageViews := make([]vk.ImageView, len(swapchainImages))

		// Create one image view per image.
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

		// return the image views
		return imageViews
	}()

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

	// Create the pipelines.
	pipelines := func() []vk.Pipeline {
		// Function for loading a shader.
		loadShaderModule := func(fn string) vk.ShaderModule {
			// load the shader bytes
			shaderWords := NewWordsUint32(MustReadFile(fn))

			// Create the info object.
			shaderInfo := vk.ShaderModuleCreateInfo{
				SType:    vk.StructureTypeShaderModuleCreateInfo,
				CodeSize: shaderWords.Sizeof(),
				PCode:    []uint32(shaderWords),
			}

			// Create the result object.
			var shaderModule vk.ShaderModule

			// Call the Vulkan function.
			MustSucceed(vk.CreateShaderModule(app.device, &shaderInfo, nil, &shaderModule))

			// return the handle
			return shaderModule
		}

		// Create the vertex shader
		vertShaderModule := loadShaderModule("shaders/vert.spv")
		defer vk.DestroyShaderModule(app.device, vertShaderModule, nil)

		// Create the fragment shader
		fragShaderModule := loadShaderModule("shaders/frag.spv")
		defer vk.DestroyShaderModule(app.device, fragShaderModule, nil)

		// Create the ShaderStage info objects.
		shaderStages := []vk.PipelineShaderStageCreateInfo{
			vk.PipelineShaderStageCreateInfo{
				SType:  vk.StructureTypePipelineShaderStageCreateInfo,
				Stage:  vk.ShaderStageVertexBit,
				Module: vertShaderModule,
				PName:  ToCString("main"),
			},
			vk.PipelineShaderStageCreateInfo{
				SType:  vk.StructureTypePipelineShaderStageCreateInfo,
				Stage:  vk.ShaderStageFragmentBit,
				Module: fragShaderModule,
				PName:  ToCString("main"),
			},
		}

		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				SType:      vk.StructureTypeGraphicsPipelineCreateInfo,
				StageCount: uint32(len(shaderStages)),
				PStages:    shaderStages,
				PVertexInputState: &vk.PipelineVertexInputStateCreateInfo{
					SType:                           vk.StructureTypePipelineVertexInputStateCreateInfo,
					VertexBindingDescriptionCount:   0,
					VertexAttributeDescriptionCount: 0,
				},
				PInputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
					SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
					Topology:               vk.PrimitiveTopologyTriangleList,
					PrimitiveRestartEnable: vk.False,
				},
				PViewportState: &vk.PipelineViewportStateCreateInfo{
					SType:         vk.StructureTypePipelineViewportStateCreateInfo,
					ViewportCount: 1,
					PViewports: []vk.Viewport{
						vk.Viewport{
							Width:    float32(extent.Width),
							Height:   float32(extent.Height),
							MaxDepth: 1.0,
						},
					},
					ScissorCount: 1,
					PScissors: []vk.Rect2D{
						vk.Rect2D{
							Offset: vk.Offset2D{},
							Extent: extent,
						},
					},
				},
				PRasterizationState: &vk.PipelineRasterizationStateCreateInfo{
					SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
					DepthClampEnable:        vk.False,
					RasterizerDiscardEnable: vk.False,
					PolygonMode:             vk.PolygonModeFill,
					LineWidth:               1.0,
					CullMode:                vk.CullModeFlags(vk.CullModeBackBit),
					FrontFace:               vk.FrontFaceClockwise,
					DepthBiasEnable:         vk.False,
				},
				PMultisampleState: &vk.PipelineMultisampleStateCreateInfo{
					SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
					SampleShadingEnable:  vk.False,
					RasterizationSamples: vk.SampleCount1Bit,
				},
				PColorBlendState: &vk.PipelineColorBlendStateCreateInfo{
					SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
					LogicOpEnable:   vk.False,
					LogicOp:         vk.LogicOpCopy,
					AttachmentCount: 1,
					PAttachments: []vk.PipelineColorBlendAttachmentState{
						vk.PipelineColorBlendAttachmentState{
							ColorWriteMask: vk.ColorComponentFlags(vk.ColorComponentRBit | vk.ColorComponentGBit | vk.ColorComponentBBit | vk.ColorComponentABit),
							BlendEnable:    vk.False,
						},
					},
				},
				Layout:     pipelineLayout,
				RenderPass: renderPass,
				Subpass:    0,
			},
		}

		// Create the result object.
		pipelines := make([]vk.Pipeline, len(pipelineInfos))

		// Call the Vulkan function.
		MustSucceed(vk.CreateGraphicsPipelines(app.device,
			vk.PipelineCache(vk.NullHandle),
			1,
			pipelineInfos,
			nil,
			pipelines))

		// Return the pipelines.
		return pipelines
	}()

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

		// Return the command buffers.
		return buffers
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
	vk.DestroyPipelineLayout(device, pipeline.PipelineLayout, nil)
	vk.DestroyRenderPass(device, pipeline.RenderPass, nil)
	for _, imgView := range pipeline.SwapchainImageViews {
		vk.DestroyImageView(device, imgView, nil)
	}
	vk.DestroySwapchain(device, pipeline.Swapchain, nil)
}
