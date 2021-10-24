package main

import (
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
	window   *glfw.Window
	instance vk.Instance
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
			EnabledExtensionCount:   0,
			PpEnabledExtensionNames: []string{},
			EnabledLayerCount:       0,
			PpEnabledLayerNames:     []string{},
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
	app := TriangleApplication{}
	app.Run()
}
