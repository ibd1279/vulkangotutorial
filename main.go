package main

import (
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

const (
	WindowWidth  = 800
	WindowHeight = 600
)

type TriangleApplication struct {
	window *glfw.Window
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

	initVulkan := func() {}
	createInstance := func() {}
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
