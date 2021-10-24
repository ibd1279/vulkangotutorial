package main

import (
	"fmt"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

func init() {
	runtime.LockOSThread()
}

const (
	WindowWidth  = 800
	WindowHeight = 600
)

type TriangleApplication struct {
	window                         *glfw.Window
	instance                       vk.Instance
	RequiredInstanceExtensionNames []string
	RequiredInstanceLayerNames     []string

	surface                   vk.Surface
	physicalDevice            PhysicalDevice
	SelectPhysicalDeviceIndex func([]PhysicalDevice, vk.Surface) int

	RequiredDeviceExtensionNames []string
	RequiredDeviceLayerNames     []string
	device                       vk.Device
	graphicsQueue                vk.Queue
	presentationQueue            vk.Queue
}

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

		// We aren't yet ready to handle resizable windows.
		glfw.WindowHint(glfw.Resizable, glfw.False)

		// Create the window object.
		app.window, err = glfw.CreateWindow(WindowWidth, WindowHeight, "Vulkan", nil, nil)
		if err != nil {
			panic(err)
		}

		// Update required extensions.
		app.RequiredInstanceExtensionNames = append(
			app.RequiredInstanceExtensionNames,
			app.window.GetRequiredInstanceExtensions()...,
		)
	}

	initVulkan := func() {
		// Link Vulkan and GLFW
		vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())

		// Initialize Vulkan
		if err := vk.Init(); err != nil {
			panic(err)
		}
	}

	createInstance := func() {
		// Available Instance Layers.
		layerProps := EnumerateInstanceLayerProperties()
		availLayerNames, availLayerDescs := LayerPropertiesNamesAndDescriptions(layerProps)
		for h := 0; h < len(layerProps); h++ {
			fmt.Printf("Layer Avail: %s %s\n",
				availLayerNames[h],
				availLayerDescs[h])
		}

		// Required Instance Layers.
		reqLayerNames := ToCStrings(DedupeSlice(app.RequiredInstanceLayerNames))
		MustSupport(availLayerNames, reqLayerNames)

		// Available Instance Extensions.
		layerExts := EnumerateInstanceExtensionProperties("")
		availExtNames := ExtensionPropertiesNames(layerExts)
		for h := 0; h < len(layerExts); h++ {
			fmt.Printf("Extension Avail: %s\n",
				availExtNames[h])
		}

		// Required Instance Extensions.
		reqExtNames := ToCStrings(DedupeSlice(app.RequiredInstanceExtensionNames))
		MustSupport(availExtNames, reqExtNames)

		// Create the info object.
		instanceInfo := vk.InstanceCreateInfo{
			SType: vk.StructureTypeInstanceCreateInfo,
			PApplicationInfo: &vk.ApplicationInfo{
				SType:              vk.StructureTypeApplicationInfo,
				PApplicationName:   ToCString("Hello Triangle"),
				ApplicationVersion: vk.MakeVersion(1, 0, 0),
				PEngineName:        ToCString("No Engine"),
				EngineVersion:      vk.MakeVersion(1, 0, 0),
				ApiVersion:         vk.ApiVersion11,
			},
			EnabledExtensionCount:   uint32(len(reqExtNames)),
			PpEnabledExtensionNames: reqExtNames,
			EnabledLayerCount:       uint32(len(reqLayerNames)),
			PpEnabledLayerNames:     reqLayerNames,
		}

		// Create the result object.
		var instance vk.Instance

		// Call the Vulkan function.
		MustSucceed(vk.CreateInstance(&instanceInfo, nil, &instance))

		// Update the application.
		app.instance = instance

		// InitInstance is required for macOs?
		vk.InitInstance(app.instance)
	}

	createSurface := func() {
		// Get the surface from the Window.
		surface, err := app.window.CreateWindowSurface(app.instance, nil)
		if err != nil {
			panic(err)
		}

		// Store the handle
		app.surface = vk.SurfaceFromPointer(surface)
	}

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
			availExtNames := ExtensionPropertiesNames(
				phyDev.ExtensionProperties,
			)

			// Calculate missing extensions.
			missingExtNames := SetSubtraction(
				app.RequiredDeviceExtensionNames,
				SliceToMap(availExtNames),
			)

			// Add supported devices.
			if len(missingLayerNames) == 0 && len(missingExtNames) == 0 {
				filteredPhysicalDevices = append(
					filteredPhysicalDevices,
					phyDev,
				)
			}
		}
		physicalDevices = filteredPhysicalDevices

		// fail if we have zero of them.
		if len(physicalDevices) == 0 {
			panic(fmt.Errorf("failed to find GPUs with Vulkan support!"))
		}

		// Ask the application to select a device.
		idx := app.SelectPhysicalDeviceIndex(physicalDevices,
			app.surface)
		if idx >= 0 && idx < len(physicalDevices) {
			app.physicalDevice = physicalDevices[idx]
		} else {
			panic(fmt.Errorf("failed to select a physical device, got index %d", idx))
		}
	}

	createLogicalDevice := func() {
		// Calculate the number of queue info structs.
		gIdx, pIdx := app.physicalDevice.QueueFamilies(app.surface)
		queueFamilyIndices := []uint32{gIdx.Val(), pIdx.Val()}
		if gIdx.Val() == pIdx.Val() {
			queueFamilyIndices = queueFamilyIndices[:1]
		}

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

	createCommandPool := func() {}
	createSemaphores := func() {}
	createFences := func() {}

	// Calls
	createWindow()
	initVulkan()
	createInstance()
	createSurface()
	pickPhysicalDevice()
	createLogicalDevice()
	createCommandPool()
	app.recreatePipeline()
	createSemaphores()
	createFences()
}

func (app *TriangleApplication) mainLoop() {
	for !app.window.ShouldClose() {
		glfw.PollEvents()
		app.drawFrame()
	}
}

func (app *TriangleApplication) drawFrame() {}

func (app *TriangleApplication) recreatePipeline() {}

func (app *TriangleApplication) cleanup() {
	vk.DestroyDevice(app.device, nil)
	vk.DestroySurface(app.instance, app.surface, nil)
	vk.DestroyInstance(app.instance, nil)
	app.window.Destroy()
	glfw.Terminate()
}

func (app *TriangleApplication) Run() {
	app.setup()
	defer app.cleanup()
	app.mainLoop()
}

func main() {
	app := TriangleApplication{
		RequiredInstanceExtensionNames: []string{},
		RequiredInstanceLayerNames: []string{
			"VK_LAYER_KHRONOS_validation",
		},
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
		RequiredDeviceLayerNames: []string{
			"VK_LAYER_KHRONOS_validation",
		},
		RequiredDeviceExtensionNames: []string{
			"VK_KHR_portability_subset",
		},
	}
	app.Run()
}

// LayerProperties
func EnumerateInstanceLayerProperties() []vk.LayerProperties {
	// Allocate the count.
	var count uint32

	// Call to get the count.
	vk.EnumerateInstanceLayerProperties(&count, nil)

	// Allocate to store the data.
	list := make([]vk.LayerProperties, count)

	// Call to get the data.
	vk.EnumerateInstanceLayerProperties(&count, list)

	// Dereference the data.
	for k, _ := range list {
		list[k].Deref()
	}

	// Return the result.
	return list
}

// ExtensionProperties
func EnumerateInstanceExtensionProperties(layerName string) []vk.ExtensionProperties {
	// Allocate the count.
	var count uint32

	// Call to get the count.
	vk.EnumerateInstanceExtensionProperties(layerName, &count, nil)

	// Allocate to store the data.
	list := make([]vk.ExtensionProperties, count)

	// Call to get the data.
	vk.EnumerateInstanceExtensionProperties(layerName, &count, list)

	// Dereference the data.
	for k, _ := range list {
		list[k].Deref()
	}

	// Return the result.
	return list
}

// Properties to Strings
func LayerPropertiesNamesAndDescriptions(props []vk.LayerProperties) ([]string, []string) {
	names, descs := make([]string, len(props)), make([]string, len(props))

	for k, p := range props {
		names[k] = vk.ToString(p.LayerName[:])
		descs[k] = vk.ToString(p.Description[:])
	}

	return names, descs
}

func ExtensionPropertiesNames(props []vk.ExtensionProperties) []string {
	names := make([]string, len(props))

	for k, p := range props {
		names[k] = vk.ToString(p.ExtensionName[:])
	}

	return names
}

// Physical Device
type PhysicalDevice struct {
	Handle                vk.PhysicalDevice
	Properties            vk.PhysicalDeviceProperties
	Features              vk.PhysicalDeviceFeatures
	LayerProperties       []vk.LayerProperties
	ExtensionProperties   []vk.ExtensionProperties
	QueueFamilyProperties []vk.QueueFamilyProperties
}

func EnumeratePhysicalDevices(instance vk.Instance) []PhysicalDevice {
	// 2-call enumerate the devices
	var count uint32
	vk.EnumeratePhysicalDevices(instance, &count, nil)
	list := make([]vk.PhysicalDevice, count)
	vk.EnumeratePhysicalDevices(instance, &count, list)

	// Loop over each device and get extra data.
	physicalDevices := make([]PhysicalDevice, len(list))
	for k, phyDev := range list {
		// Store the Handle.
		physicalDevices[k].Handle = phyDev

		// Get the physical device properties.
		vk.GetPhysicalDeviceProperties(phyDev, &physicalDevices[k].Properties)

		// Get the physical device features.
		vk.GetPhysicalDeviceFeatures(phyDev, &physicalDevices[k].Features)

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

	// return the result.
	return physicalDevices
}

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
