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

func (pipeline *Pipeline) Cleanup(device vk.Device) {
	for _, buffer := range pipeline.SwapchainFramebuffers {
		vk.DestroyFramebuffer(device, buffer, nil)
	}

	vk.DestroyPipelineLayout(device, pipeline.PipelineLayout, nil)
	vk.DestroyRenderPass(device, pipeline.RenderPass, nil)
	for _, imgView := range pipeline.SwapchainImageViews {
		vk.DestroyImageView(device, imgView, nil)
	}
	vk.DestroySwapchain(device, pipeline.Swapchain, nil)
}
