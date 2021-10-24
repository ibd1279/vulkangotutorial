# Vulcan, Go, and A Triangle, Part 5

In this part I am going to create an object for keeping track of our physical device, enumerate over physical devices, and select a physical device for our application.

I deviate from the vulkan tutorial here a little bit because I wanted to encapsulate physical device related functionality in a specific class. This will become more useful later when dealing with memory buffers. I also create the surface in a different order.

This part relates to [Drawing a triangle / Setup / Physical devices and queue families](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Physical_devices_and_queue_families) in the original tutorial.

<!--more-->

## Physical vs Logical

Vulkan has a concept for PhysicalDevice and a concept for Device. The physical device is exactly that -- a physical device. Think of it as the physical GPU or graphics card in my system. Each physical card requires specific drivers and capabilities. It is passed around using the `vk.PhysicalDevice` handle. As a result, I explicitly refer to it as a physical device.

The logical device is the Vulkan concept for a physical device. It represents a more abstracted definition of a device and allows your program to communicate in vulkan abstract terms rather than device and driver specific terms. The application will pass around a handle to the logical device through a `vk.Device` handle. As a result, I generally drop the "logical" and just refer to it as a device.

## Describing a physical device

In my implementation, I found it easier to load and cache the physical device details into a dedicated structure. This helped some buffer aspects much later in the tutorial; for example, aligning the uniform buffer object memory. It also allowed me to deal with putting all the `Deref()` calls in a single place.

I created a new type at the bottom of the main file.

```golang
// Physical Device
type PhysicalDevice struct {
	Handle                vk.PhysicalDevice
	Properties            vk.PhysicalDeviceProperties
	Features              vk.PhysicalDeviceFeatures
	LayerProperties       []vk.LayerProperties
	ExtensionProperties   []vk.ExtensionProperties
	QueueFamilyProperties []vk.QueueFamilyProperties
}
```

The handle is the physical device handle used in Vulkan calls. The properties and features fields are informative structs that will be used in physical device selection.

The layer properties and extension properties are similar to the instance layer and extension properties, but device specific. I will be adding the ability to require them later in this part.

Queue family properties represents the properties of the different processing queues on the physical device.

Under the new type, I created a function for enumerating physical devices, similar to how we enumerated layer properties.

```golang
func EnumeratePhysicalDevices(instance vk.Instance) []PhysicalDevice {
	// 2-call enumerate the devices
		
	// Loop over each device and get extra data.
	
	// return the result.
}
```

For this function, I'm going to collapse down the number of comments as the pattern was already introduced in the previous part.

```golang
func EnumeratePhysicalDevices(instance vk.Instance) []PhysicalDevice {
	// 2-call enumerate the devices
	var count uint32
	vk.EnumeratePhysicalDevices(instance, &count, nil)
	list := make([]vk.PhysicalDevice, count)
	vk.EnumeratePhysicalDevices(instance, &count, list)
	...
}
```

We are going to be making additional enumeration and get calls inside the for-loop.

```golang
func EnumeratePhysicalDevices(instance vk.Instance) []PhysicalDevice {
	...
	// Loop over each device and get extra data.
	physicalDevices := make([]PhysicalDevice, len(list))
  	for k, phyDev := range list {
		// Store the Handle.
		physicalDevices[k].Handle = phyDev

		// Get the physical device properties.

		// Get the physical device features.
		
		// 2-call enumerate the layer properties.

		// 2-call enumerate the extension properties.

		// 2-call enumerate the queue family properties.

		// Dereference the data.
	}
	
	// return the result.
	return physicalDevices
}
```

The calls here are pretty straight forward, and I'm not doing much but pulling the data into Go for easier access later.

```golang
func EnumeratePhysicalDevices(instance vk.Instance) []PhysicalDevice {
	...
  	for k, phyDev := range list {
		...
		// Get the physical device properties.
		vk.GetPhysicalDeviceProperties(phyDev, &physicalDevices[k].Properties)

		// Get the physical device features.
		vk.GetPhysicalDeviceFeatures(phyDev, &physicalDevices[k].Features)
		...
	}
	...
}
```

The first two are straightforward calls to Vulkan. The properties call is useful for getting the physical device name, type, and limits. The features call gets a list of boolean flags that denote what the physical device is capable of supporting.

For the remaining 2-call enumerations, I will reuse the previous defined count variable.

```golang
func EnumeratePhysicalDevices(instance vk.Instance) []PhysicalDevice {
	...
  	for k, phyDev := range list {
	  	...
		// 2-call enumerate the layer properties.
		vk.EnumerateDeviceLayerProperties(phyDev, &count, nil)
		physicalDevices[k].LayerProperties = make([]vk.LayerProperties, count)
		vk.EnumerateDeviceLayerProperties(phyDev, &count, physicalDevices[k].LayerProperties)

		// 2-call enumerate the extension properties.
		vk.EnumerateDeviceExtensionProperties(phyDev, "", &count, nil)
		physicalDevices[k].ExtensionProperties = make([]vk.ExtensionProperties, count)
		vk.EnumerateDeviceExtensionProperties(phyDev, "", &count, physicalDevices[k].ExtensionProperties)

		// 2-call enumerate the queue family properties.
		vk.GetPhysicalDeviceQueueFamilyProperties(phyDev, &count, nil)
		physicalDevices[k].QueueFamilyProperties = make([]vk.QueueFamilyProperties, count)
		vk.GetPhysicalDeviceQueueFamilyProperties(phyDev, &count, physicalDevices[k].QueueFamilyProperties)
		...
	}
	...
}
```

While it is possible to have different device extensions per layer, this tutorial skips over that.

Finally, I dereference all these objects so that the rest of the application can ignore this aspect.

```golang
func EnumeratePhysicalDevices(instance vk.Instance) []PhysicalDevice {
	...
  	for k, phyDev := range list {
	  	...
		// Dereference the data.
		physicalDevices[k].Properties.Deref()
		physicalDevices[k].Properties.Limits.Deref()
		physicalDevices[k].Features.Deref()
		for h := 0; h < len(physicalDevices[k].LayerProperties); h++ {
			physicalDevices[k].LayerProperties[h].Deref()
		}
		for h := 0; h < len(physicalDevices[k].ExtensionProperties); h++ {
			physicalDevices[k].ExtensionProperties[h].Deref()
		}
		for h := 0; h < len(physicalDevices[k].QueueFamilyProperties); h++ {
			physicalDevices[k].QueueFamilyProperties[h].Deref()
		}
	}
	...
}
```

In a more memory constrained environment, I'd probably move these into methods instead of populating the struct, but most of this data will get thrown away after we select a device.

I implemented the stringer interface on physical devices to make it easier to output device information.

```golang
func (phyDev PhysicalDevice) String() string {
	devName := vk.ToString(phyDev.Properties.DeviceName[:])

	devType := "other"
	switch phyDev.Properties.DeviceType {
	case vk.PhysicalDeviceTypeIntegratedGpu:
		devType = "Integrated GPU"
		break
	case vk.PhysicalDeviceTypeDiscreteGpu:
		devType = "Discrete GPU"
		break
	case vk.PhysicalDeviceTypeVirtualGpu:
		devType = "Virtual GPU"
		break
	case vk.PhysicalDeviceTypeCpu:
		devType = "CPU"
		break
	}

	queueFamilyFlags := make([]string, len(phyDev.QueueFamilyProperties))
	for h := 0; h < len(phyDev.QueueFamilyProperties); h++ {
		queueFamilyFlags[h] = fmt.Sprintf("%d={flags: %05b}",
			h,
			phyDev.QueueFamilyProperties[h].QueueFlags)
	}

	return fmt.Sprintf("%s(%s) QueueFamilies:%v",
		devName, devType,
		queueFamilyFlags,
	)
}
```

The device type is informative for debugging. According to the [Vulkan specification](https://www.khronos.org/registry/vulkan/specs/1.2-extensions/man/html/VkPhysicalDeviceType.html), "The physical device type is advertised for informational purposes only, and does not directly affect the operation of the system."  Here I am outputting it for debugging purposes.

The other interesting thing here is the QueueFlags. These are bit packed flags representing a queue families capabilities. We are going to be using these to select a physical device in a bit, and displaying the flags here helped me with debugging.

I ignored the properties and features fields because they contain TONs of fields, and it made it difficult to find anything in the output.

I output the list of devices at the start of pick physical device to see my physical devices.

```golang
func (app *TriangleApplication) setup() {
	...
	createSurface := func() {}
	pickPhysicalDevice := func() {
		// Output all the physical devices.
		physicalDevices := EnumeratePhysicalDevices(app.instance)
		for k, phyDev := range physicalDevices {
			fmt.Printf("Physical Device Avail %d: %v\n", k, phyDev)
		}

		// fail if we have zero of them.
		if len(physicalDevices) == 0 {
			panic(fmt.Errorf("failed to find GPUs with Vulkan support!"))
		}
	}
	...
}
```

I also added a simple check to make sure we have at least one physical device before we continue.

When I ran the application at this point, I was able to see my two GPUs on my development machine: one discrete and one integrated.

## Creating the surface.

Before I can pick a physical device, I need a vulkan handle to the presentation surface. This is because the presentation surface has a direct impact on the capabilities required by the application.

Creating a surface is straight forward and starts with adding a field to the application.

```golang
type TriangleApplication struct {
	...
	
	surface vk.Surface
}
```

Then we get a surface handle from GLFW.

```golang
func (app *TriangleApplication) setup() {
	...
	createSurface := func() {
		// Get the surface from the Window.
		surface, err := app.window.CreateWindowSurface(app.instance, nil)
		if err != nil {
			panic(err)
		}

		// Store the handle
		app.surface = vk.SurfaceFromPointer(surface)
	}
	...
}
```

The Vulkan Tutorial for [Window surface](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Window_surface) provides a description of what is happening behind the scenes

And finally, destroy the surface handle right before destroying the instance.

```golang
func (app *TriangleApplication) cleanup() {
	vk.DestroySurface(app.instance, app.surface, nil)
	...
}
```

## Intersectional requirements

Physical device selection is based on the intersection of what the application intends to use the device for and what the presentation service needs. The official tutorial goes through several different examples to explain the many different ways an application could customize this, but I am going to do something simple.

I started by extending my PhysicalDevice class to generate the intersection between the surface and the physical device. These methods were added after the String method.

```golang
func (phyDev PhysicalDevice) QueueFamilies(surface vk.Surface) (graphics, presentation OptionUint32) {
	// Iterate over Queue Families to find support.
}

func (phyDev PhysicalDevice) SwapchainSupport(surface vk.Surface) (capabilities vk.SurfaceCapabilities, formats []vk.SurfaceFormat, presentModes []vk.PresentMode) {
	// Get the intersection of capabilities.

	// 2-call enumerate the formats.
	
	// 2-call enumerate the present modes.
} 
```

The queue families function is mainly checking the queue family flags to see if it supports graphics and presentation to the provided surface.

```golang
func (phyDev PhysicalDevice) QueueFamilies(surface vk.Surface) (graphics, presentation OptionUint32) {
	// Iterate over Queue Families to find support.
	for k, v := range phyDev.QueueFamilyProperties {
		// cast as everything is expecting a uint32
		index := uint32(k)

		// Check if the queue supports graphics commands.
		if v.QueueFlags&vk.QueueFlags(vk.QueueGraphicsBit) != 0 {
			graphics.Set(index)
		}

		// check if this physical device can draw to our surface.
		var presentSupport vk.Bool32
		vk.GetPhysicalDeviceSurfaceSupport(
			phyDev.Handle,
			index,
			surface,
			&presentSupport,
		)
		if presentSupport.B() {
			presentation.Set(index)
		}

		// If both families have values, we can stop iteration.
		if graphics.IsSet() && presentation.IsSet() {
			break
		}
	}
	return graphics, presentation
}
```

The method uses flags to check if a physical device's queue family supports graphics commands.  Then it checks if a device is able to present to our surface.  Once it has values for both -- preferably on the same queue family, but fine if they aren't -- it returns the values.

The Swapchain support, which will be used more in pipeline creation, ensures that the modes and formats of the device are compatible with the modes and formats of the surface.

```golang
func (phyDev PhysicalDevice) SwapchainSupport(surface vk.Surface) (capabilities vk.SurfaceCapabilities, formats []vk.SurfaceFormat, presentModes []vk.PresentMode) {
	// Get the intersection of capabilities.
	vk.GetPhysicalDeviceSurfaceCapabilities(phyDev.Handle,
		surface,
		&capabilities)
	capabilities.Deref()
	capabilities.CurrentExtent.Deref()
	capabilities.MinImageExtent.Deref()
	capabilities.MaxImageExtent.Deref()

	// 2-call enumerate the formats.
	var count uint32
	vk.GetPhysicalDeviceSurfaceFormats(phyDev.Handle,
		surface,
		&count,
		nil)
	formats = make([]vk.SurfaceFormat, count)
	vk.GetPhysicalDeviceSurfaceFormats(phyDev.Handle,
		surface,
		&count,
		formats)
	for k, _ := range formats {
		formats[k].Deref()
	}

	// 2-call enumerate the present modes.
	vk.GetPhysicalDeviceSurfacePresentModes(phyDev.Handle,
		surface,
		&count,
		nil)
	presentModes = make([]vk.PresentMode, count)
	vk.GetPhysicalDeviceSurfacePresentModes(phyDev.Handle,
		surface,
		&count,
		presentModes)

	return capabilities, formats, presentModes
}
```

This is mostly aggregating API calls together (and centralizing the Deref() calls). The decisions on this data will happen in the next section.

## Pick me Physical Device

I pushed device creation into a signature for my implementation. This was to illustrate that device selection is really up to the requirements of the application and was not something Vulkan or a framework could abstract.

I created a field for the selected physical device and the selection function.

```golang
type TriangleApplication struct {
	...
	surface                   vk.Surface
	physicalDevice            PhysicalDevice
	SelectPhysicalDeviceIndex func([]PhysicalDevice, vk.Surface) int
}
```

The SelectPhysicalDeviceIndex function will take a slice of physical devices and a surface. It returns the index of the physical device it wants to use. It returns a negative number if no device is acceptable.

With this API contract, I can update the selectPhysicalDevice function.

```golang
func (app *TriangleApplication) setup() {
	...
	pickPhysicalDevice := func() {
		...
		// Ask the application to select a device.
		idx := app.SelectPhysicalDeviceIndex(physicalDevices,
			app.surface)
		if idx >= 0 && idx < len(physicalDevices) {
			app.physicalDevice = physicalDevices[idx]
		} else {
			panic(fmt.Errorf("failed to select a physical device, got index %d", idx))
		}
	}
	...
}
```

Then I provided a selection function in the Application initialization.

```golang
func main() {
	app := TriangleApplication{
		...
		SelectPhysicalDeviceIndex: func(physicalDevices []PhysicalDevice, surface vk.Surface) int {
			// Select a device
			for k, phyDev := range physicalDevices {
				gIdx, pIdx := phyDev.QueueFamilies(surface)
				_, fmts, modes := phyDev.SwapchainSupport(surface)
				if gIdx.IsSet() && pIdx.IsSet() && len(fmts) > 0 && len(modes) > 0 {
					fmt.Printf("Physical Device Selected: %d %s\n",
						k,
						phyDev)
					return k
				}
			}
			return -1
		},
	}
	...
}
```

This method calls both of the new functions, first to get the queue family indices, then to get the supported modes. If we have values for all 4 things, we can use this device. If we have a failure, we return -1. This is effectively a "first that does everything" approach to device selection.

I was able to run the application now and the application selected my discrete GPU.

We don't have to clean up after the physical device. Destroying that would generally make people unhappy, at least that is the impression I got from the NewWorld beta news.

## Conclusion

This part introduced the physical device and the surface. It also introduced the intersections between the two concepts as a method for selecting which physical device to use. Our implementation selects the first that works. You can extend this approach by looking at the options suggested by The [Vulkan Tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Physical_devices_and_queue_families#page_Base-device-suitability-checks). 

The next part will create the logical device, with certain required extensions.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/e875e513d2c94a5becc08fb357c7eb2c795ff818/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/e875e513d2c94a5becc08fb357c7eb2c795ff818#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261))
