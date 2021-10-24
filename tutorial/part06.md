# Vulcan, Go, and A Triangle, Part 6

In this part we are going to add support for required device layers and extensions before creating a logical device.  The logical device creation process is similar to the instance creation process. Instead of telling Vulkan about our application, we will be telling Vulkan about our device requirements.

This part follows closely with the Vulkan Tutorial. I do push the required extensions checks into the device selection function, but otherwise the steps are similar to [Drawing a triangle / Setup / Logical device and queues](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Logical_device_and_queues).

<!--more-->

## Device layers and extensions

Similar to the instance, you can specify layers and extensions to enable on the device.

You technically don't need to deal with the device layers. From the [Vulkan Tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Logical_device_and_queues#page_Creating-the-logical-device):

> "Previous implementations of Vulkan made a distinction between instance and device specific validation layers, but this is no longer the case. That means that the enabledLayerCount and ppEnabledLayerNames fields of VkDeviceCreateInfo are ignored by up-to-date implementations. However, it is still a good idea to set them anyway to be compatible with older implementations".

I added two new fields for device specific layers and extensions.

```golang
type TriangleApplication struct {
	...

	RequiredDeviceExtensionNames []string
	RequiredDeviceLayerNames     []string
}
```

And I already knew that I wanted to enable the validation layers, so I added that to the main function.

```golang
func main() {
	app := TriangleApplication{
		...
		RequiredDeviceLayerNames: []string{
			"VK_LAYER_KHRONOS_validation",
		},
		RequiredDeviceExtensionNames: []string{},
	}
	...
}
```

I added the filtering to the pickPhysicalDevice function, as that seemed an appropriate place to filter out devices that wouldn't work.

```golang
func (app *TriangleApplication) setup() {
	...
	pickPhysicalDevice := func() {
		// Output all the physical devices.
		physicalDevices := EnumeratePhysicalDevices(app.instance)
		for k, phyDev := range physicalDevices {
			fmt.Printf("Physical Device Avail %d: %v\n", k, phyDev)
		}

		// Filter devices based on required support.
		filteredPhysicalDevices := make([]PhysicalDevice, 0, len(physicalDevices))
		for _, phyDev := range physicalDevices {
			// Get device layer support.
			availLayerNames, _ := LayerPropertiesNamesAndDescriptions(
				phyDev.LayerProperties,
			)

			// Calculate missing layers.
			missingLayerNames := SetSubtraction(
				app.RequiredDeviceLayerNames,
				SliceToMap(availLayerNames),
			)

			// Get device extension support.

			// Calculate missing extensions.

			// Add supported devices.
			
		}
		physicalDevices = filteredPhysicalDevices

		// fail if we have zero of them.
		if len(physicalDevices) == 0 {
			panic(fmt.Errorf("failed to find GPUs with Vulkan support!"))
		}
		...
	}
	...
}
```

I created a new slice to collect all the devices that will work. Then I looped over the physical devices, getting the available layer names, calculating a list of missing names using a helper function we added in part 1. After the for loop, I reset the physicalDevices list.

I repeated the missing logic again for the extensions.

```golang
func (app *TriangleApplication) setup() {
	...
	pickPhysicalDevice := func() {
		...
		for _, phyDev := range physicalDevices {
			...
			// Get device extension support.
			availExtNames := ExtensionPropertiesNames(
				phyDev.ExtensionProperties,
			)

			// Calculate missing extensions.
			missingExtNames := SetSubtraction(
				app.RequiredDeviceExtensionNames,
				SliceToMap(availExtNames),
			)
			...
		}
		...
	}
	...
}
```

If nothing is missing from either category, I add the device to the filtered list.

```golang
func (app *TriangleApplication) setup() {
	...
	pickPhysicalDevice := func() {
		...
		for _, phyDev := range physicalDevices {
			...
			// Add supported devices.
			if len(missingLayerNames) == 0 && len(missingExtNames) == 0 {
				filteredPhysicalDevices = append(
					filteredPhysicalDevices,
					phyDev,
				)
			}
		}
		...
	}
	...
}
```

At this point, my SelectPhysicalDeviceIndex function would never receive a device missing required layers or extensions.

## Creating the (logical) device and graphics queue

I had all of the inputs and processing required to create my logical device. I'll generally be referring to it as a "device" going forward, and lazily dropping the "logical".

I started by adding a field to store the device handle and the queues.

```golang
type TriangleApplication struct {
	...
	device                       vk.Device
	graphicsQueue                vk.Queue
	presentationQueue            vk.Queue
}
```

Then I did a (more complicated) version of the creation pattern introduced in part 3.

```golang
func (app *TriangleApplication) setup() {
	...
	createLogicalDevice := func() {
		// Calculate the number of queue info structs.

		// Populate the queue infos.

		// Create the info object.

		// Create the result object.

		// Call the Vulkan function.

		// Update the application.

		// Fetch the graphics queue handle.

		// Fetch the presentation queue handle.
	}
	...
}
```

While it was not true for my hardware, it is possible that the graphics and the presentation queues are different queue families. It is also possible they are the same.

In order to work around this, I created the following code to normalize everything.

```golang
func (app *TriangleApplication) setup() {
	...
	createLogicalDevice := func() {
		// Calculate the number of queue info structs.
		gIdx, pIdx := app.physicalDevice.QueueFamilies(app.surface)
		queueFamilyIndices := []uint32{gIdx.Val(), pIdx.Val()}
		if gIdx.Val() == pIdx.Val() {
			queueFamilyIndices = queueFamilyIndices[:1]
		}
		...
	}
	...
}
```

If the graphics index and presentation index are the same, we reduce the slice size to 1.  Everything else from here assumes that the head of the slice is the graphics queue family and the tail of the slice is the presentation family.

Next came the queue info structs themselves.

```golang
func (app *TriangleApplication) setup() {
	...
	createLogicalDevice := func() {
		...
		// Populate the queue infos.
		queueCreateInfos := make([]vk.DeviceQueueCreateInfo, len(queueFamilyIndices))
		for k, idx := range queueFamilyIndices {
			queueCreateInfos[k] = vk.DeviceQueueCreateInfo{
				SType:            vk.StructureTypeDeviceQueueCreateInfo,
				QueueFamilyIndex: idx,
				QueueCount:       1,
				PQueuePriorities: []float32{1.0},
			}
		}
		...
	}
	...
}
```

I cannot say much here that would be word-for-word what is on the [Vulkan Tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Logical_device_and_queues#page_Specifying-the-queues-to-be-created). I suggest reading there.

Then came the info / result / call vulkan pattern itself:

```golang
func (app *TriangleApplication) setup() {
	...
	createLogicalDevice := func() {
		...
		// Create the info object.
		deviceInfo := vk.DeviceCreateInfo{
			SType:                   vk.StructureTypeDeviceCreateInfo,
			QueueCreateInfoCount:    uint32(len(queueCreateInfos)),
			PQueueCreateInfos:       queueCreateInfos,
			EnabledLayerCount:       uint32(len(app.RequiredDeviceLayerNames)),
			PpEnabledLayerNames:     ToCStrings(app.RequiredDeviceLayerNames),
			EnabledExtensionCount:   uint32(len(app.RequiredDeviceExtensionNames)),
			PpEnabledExtensionNames: ToCStrings(app.RequiredDeviceExtensionNames),
		}

		// Create the result object.
		var device vk.Device

		// Call the Vulkan function.
		MustSucceed(vk.CreateDevice(app.physicalDevice.Handle, &deviceInfo, nil, &device))
		
		// Update the application.
		app.device = device
		...
	}
	...
}
```

I had originally added the PEnabledFeatures onto the info struct, but that caused me nothing but pain and errors. The [Vulkan Tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Logical_device_and_queues#page_Specifying-used-device-features) dedicated a subheading to it being blank, but it appears that the generated structure for the [vulkan-go is incorrect](https://github.com/vulkan-go/vulkan/issues/57).

The last thing I need to do here is get the handles to my graphics and presentation queues. Remembering that head is graphics and tail is presentation, we fetch both of them identically.

```golang
func (app *TriangleApplication) setup() {
	...
	createLogicalDevice := func() {
		...
		// Fetch the graphics queue handle.
		var queue vk.Queue
		queueIndex := 0
		vk.GetDeviceQueue(app.device, queueFamilyIndices[queueIndex], uint32(queueIndex), &queue)
		app.graphicsQueue = queue

		// Fetch the presentation queue handle.
		queueIndex = len(queueFamilyIndices) - 1
		vk.GetDeviceQueue(app.device, queueFamilyIndices[queueIndex], uint32(queueIndex), &queue)
		app.presentationQueue = queue
	}
	...
}
```

## Clean up

The queues will get cleaned up by the device clean up, so I only add the device to the cleanup function.

```golang
func (app *TriangleApplication) cleanup() {
	vk.DestroyDevice(app.device, nil)
	...
}
```

## Mac issue

When I ran the application on my Mac, I received this validation error:

> vkCreateDevice: VK_KHR_portability_subset must be enabled because physical device VkPhysicalDevice 0x4b12a00[] supports it The Vulkan spec states: If the [VK_KHR_portability_subset] extension is included in pProperties of vkEnumerateDeviceExtensionProperties, ppEnabledExtensions must include "VK_KHR_portability_subset".

The correct thing would be to make the code follow the spec guidelines automatically, but this is a tutorial, not a production application, so I skipped that, and just added "VK_KHR_portability_subset" as a RequiredDeviceExtensionNames.

```golang
func main() {
	app := TriangleApplication{
		...
		RequiredDeviceExtensionNames: []string{
			"VK_KHR_portability_subset",
		},
	}
	app.Run()
}
```

And that made the validation warning go away. I think at one point, I also had to add "VK_KHR_get_physical_device_properties2" as an instance extension, but maybe that will come up in a later part.

Part of the reason I added support for extensions and layers was to simplify dealing with this type of issue when it pops up.

## Conclusion

In this part I filtered my physical devices based on my required layers and extensions. I also created a device and the graphics and presentation queues. This puts the tutorial at the end of [Drawing a triangle / Presentation / Swap chain](https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Window_surface).

In the next part, I'll start with the pipeline and the swapchain.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/8551d04a5b52883325eb0c39d1fea247c1e65eac/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/8551d04a5b52883325eb0c39d1fea247c1e65eac#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261))
