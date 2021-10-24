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

	createSurface := func() {}
	pickPhysicalDevice := func() {}
	createLogicalDevice := func() {}
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
