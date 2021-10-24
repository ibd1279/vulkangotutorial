package main

import (
	vk "github.com/vulkan-go/vulkan"
)

type Pipeline struct {
	Swapchain            vk.Swapchain
	SwapchainImages      []vk.Image
	SwapchainImageFormat vk.Format
	SwapchainExtent      vk.Extent2D
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

	// Create and return the pipeline
	return &Pipeline{
		Swapchain:            swapchain,
		SwapchainImages:      swapchainImages,
		SwapchainImageFormat: format.Format,
		SwapchainExtent:      extent,
	}
}

func (pipeline *Pipeline) Cleanup(device vk.Device) {
	vk.DestroySwapchain(device, pipeline.Swapchain, nil)
}
