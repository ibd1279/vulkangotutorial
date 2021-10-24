# Vulcan, Go, and A Triangle, Part 7

In this part, I am going to add the for the swapchain. My application will eventually take one of these swap chain images and draw to it. The graphics card will use these images to synchronize with the refresh rate.

This part follows closely with [Drawing a triangle / Presentation / Image Views](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain). I've opted to keep swapchain a single word, regardless of my spell checker; this is mostly because Vulkan treats it as one word in the API. So expect to see `Swapchain` where the Vulkan Tutorial would have written `SwapChain`.

<!--more-->

## A different class

For my implementation, I moved all of the pipeline creation steps into a dedicated pipeline class. These are specifically all the parts that are pipeline specific or would need to be recreated when the window is resized.

I also move them into a dedicated file. This reduced the cognitive load of searching for things inside the single main file.

## Swapchain device extension

In the last part, I added support for required device extensions. The first step to setting up the swapchain is to add the swapchain as a required device extension. Vulkan provides a constant for this string.

```golang
func main() {
	app := TriangleApplication{
		...
		RequiredDeviceExtensionNames: []string{
			"VK_KHR_portability_subset",
			vk.KhrSwapchainExtensionName,
		},
	}
	app.Run()
}
```

If your device doesn't require the portability subset, feel free to exclude that one.

## The skeleton

Create the new structure and a function to create instances of it in `pipeline.go`. Similar to the `setup()` method in the Triangle application, I've listed most of the steps as comments to help explain the skeleton and make it easier to find where to add code.

```golang
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

		// Swapchain support.

		// Formats.

		// Present Mode.

		// Extent.

		// Image Count.

		// Queue Families and Share mode.

		// Create the info object.

		// Create the result object.

		// Call the Vulkan function.

		// Fetch the Swapchain Images.

		// return the swapchain and images.
	}()

	// Create and return the pipeline
	return &Pipeline{
		Swapchain:            swapchain,
		SwapchainImages:      swapchainImages,
		SwapchainImageFormat: format.Format,
		SwapchainExtent:      extent,
	}
}

func (pipeline *Pipeline) Cleanup(device vk.Device) {}
```

The initial struct contains fields for the Swapchain handle, the Image handles, the format of the swapchain images, and the the extent (dimensions) of the images.

The new pipeline function will get pretty long by the end, but it will contain all of the pipeline creation parts. We are starting with the swapchain. I've encapsulated the swapchain creation into an anonymous function to allow easier refactoring later.

## Collecting data

I'm going to focus on the code for this section. Most of this code is directly translated from the Vulkan Tutorial section on [Choosing the right settings for the swap chain](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain#page_Choosing-the-right-settings-for-the-swap-chain).

We extract the old Swapchain handle from the old pipeline, if one was provided.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		// Capture the old Swapchain.
		oldSwapchain := vk.Swapchain(vk.NullHandle)
		if oldPipeline != nil {
			oldSwapchain = oldPipeline.Swapchain
		}
		
		...
	}()
	...
}
```

I can reuse the function I created for selecting the physical device to get the support swapchain elements from the physical device.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
		// Swapchain support.
		caps, fmts, modes := app.physicalDevice.SwapchainSupport(app.surface)
		
		...
	}()
	...
}
```

Next I select a surface format.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
		// Formats.
		format := func() vk.SurfaceFormat {
			for _, v := range fmts {
				if v.Format == vk.FormatB8g8r8a8Srgb && v.ColorSpace == vk.ColorSpaceSrgbNonlinear {
					return v
				}
			}
			return fmts[0]
		}()
		
		...
	}()
	...
}
```

A [vk.SurfaceFormat](https://pkg.go.dev/github.com/vulkan-go/vulkan#SurfaceFormat) contains a Format and a ColorSpace. If we the device supports the preferred format and color space, I use that one. Otherwise I fall back to the first one in the list.

Then I select a present mode.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
		// Present Mode.
		presentMode := func() vk.PresentMode {
			for _, v := range modes {
				if v == vk.PresentModeMailbox {
					return v
				}
			}
			return vk.PresentModeFifo
		}()
	
		...
	}()
	...
}
```

Fifo is guaranteed to be available. If Mailbox is available, I'll prefer that.

Then I select the extent based on the current extent or the window framebuffer.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
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
		
		...
	}()
	...
}
```

If the current extent width is MaxUint32, the device will support whatever I need. If it has a different value, I use that value. Most of this anonymous function is using the Clamp helper from part 1 to make sure we stick with the device capabilities. More details are available in [Swap extent](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain#page_Swap-extent).

Then I did the same thing with the image count.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
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
		
		...
	}()
	...
}
```

The Vulkan Tutorial describes some of the logic behind selecting the image count in the [Creating the swap chain](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain#page_Creating-the-swap-chain) section.

Next, I got the queue family indices and share mode to use. 

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
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
	
		...
	}()
	...
}
```

Again, the Vulkan Tutorial goes pretty deep into the decisions behind these values. Check the [Creating the swap chain](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain#page_Creating-the-swap-chain) section for more details.

## Create the swapchain; get the images

Once I had all the input values, I did the create pattern to create the Swapchain: info, return, call.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
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
		
		...
	}()
	...
}
```

And I follow that up with the 2-call enumeration of the swapchain images. More details on what this is doing is available as part of [Retrieving the swap chain images](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain#page_Retrieving-the-swap-chain-images).

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
		// Fetch the Swapchain Images.
		var count uint32
		vk.GetSwapchainImages(app.device, swapchain, &count, nil)
		images := make([]vk.Image, count)
		vk.GetSwapchainImages(app.device, swapchain, &count, images)
	
		...
	}()
	...
}
```

Finally, I return the important values to the higher level scope.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	swapchain, swapchainImages, format, extent := func() (vk.Swapchain, []vk.Image, vk.SurfaceFormat, vk.Extent2D) {
		...
		// return the swapchain and images.
		return swapchain, images, format, extent
	}()

	...
}
```

## Creating Cleanup

Each pipeline object is expected to manage its own resources. So I needed to populate the Cleanup method to destroy the swapchain.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	vk.DestroySwapchain(device, pipeline.Swapchain, nil)
}
```

I did a build here, mostly to check to make sure I'd set everything up correctly.

## (Re)creating the pipeline

Going back to the main.go file, I added the calls to create and clean up the pipeline as part of the Triangle application.

Starting with adding the pipeline as a field of the application.

```golang
type TriangleApplication struct {
	...

	pipeline *Pipeline
}
```

Then I added a body to the recreatePipeline method.

```golang
func (app *TriangleApplication) recreatePipeline() {
	// Create the new pipeline.
	pipeline := NewPipeline(app, app.pipeline)

	// Destroy the old pipeline.
	if app.pipeline != nil {
		app.pipeline.Cleanup(app.device)
	}

	// Update the application.
	app.pipeline = pipeline
}
```

Most of the logic in this method is dedicated to calling Cleanup() on the old pipeline if one exists.

I also needed to add that logic to the application clean up.

```golang
func (app *TriangleApplication) cleanup() {
	if app.pipeline != nil {
		app.pipeline.Cleanup(app.device)
	}
	vk.DestroyDevice(app.device, nil)
	vk.DestroySurface(app.instance, app.surface, nil)
	vk.DestroyInstance(app.instance, nil)
	app.window.Destroy()
	glfw.Terminate()
}
```

## Conclusion

I have my rendering and presentation targets created. As I continue building out the Pipeline, I will keep adding anonymous functions for each new concept that gets added to the pipeline.

I've completed the Vulkan Tutorial up to [Drawing a triangle / Presentation / Swap chain](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain). In the next part I will create the image views, render pass, and the pipeline layout.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/06e0d61afe45de96cde0059fb55ca392a8c8f307/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/06e0d61afe45de96cde0059fb55ca392a8c8f307#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261)) / [pipeline.go](https://github.com/ibd1279/vulkangotutorial/blob/06e0d61afe45de96cde0059fb55ca392a8c8f307/pipeline.go)
