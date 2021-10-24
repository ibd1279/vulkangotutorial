# Vulcan, Go, and A Triangle, Part 2

In the last part, we started with adding dependencies, helper functions and the basic skeleton. In this part we are going to start expanding on `setup()`, `cleanup()`, and `mainLoop()`.

Each part going forward will end with code that should build and run, although in many cases there will not be a visible output.

Part 2 roughly translates to the second half of [Drawing a triangle / Setup / Base code](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Base_code#page_Integrating-GLFW).

## The setup method

Setting up our Vulkan application will be done in several steps through the course of this tutorial. We can start by populating the setup method with an enumeration of the high level steps.

```golang
func (app *TriangleApplication) setup() {
	// Steps.
	createWindow := func() {}
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
```

Each of these steps are relatively simple. They are broken into functions to help illustrate the encapsulation of a task, and provide a clear idea about the number of steps involved in setting up the pipeline.

## Integrating GLFW

GPUs can be used for many things beyond displaying to the screen. As a result, Vulkan doesn't need the window object. But in order to see the results of the tutorial, we will need to display our images somewhere.

Since vulkan doesn't provide integrations with the windowing system, we depend on GLFW for creating the window for us.

Start by importing the right GLFW:

```golang
import "github.com/go-gl/glfw/v3.3/glfw"
```

After that define some constants for the initial size of the window.

```golang
const (
	WindowWidth  = 800
	WindowHeight = 600
) 
```

We also need to capture the window pointer, as we will be using it later in the tutorial. Add a field to the ```TriangleApplication```.

```golang
type TriangleApplication struct {
	window *glfw.Window
}
```

Next, we need to populate `createWindow` inside `setup()`. In order to avoid plagiarizing the section from the [Vulkan Tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Base_code#page_Integrating-GLFW), I'll direct you there for the break down of each of these calls.

```golang
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
	
	...
}
```

We need to integrate GLFW into the cleanup method. This ensures that GLFW stops callbacks.

```golang
func (app *TriangleApplication) cleanup() {
	app.window.Destroy()
	glfw.Terminate()
}
```

The documentation for [Destroy](https://pkg.go.dev/github.com/go-gl/glfw/v3.3/glfw#Window.Destroy) and [Terminate](https://pkg.go.dev/github.com/go-gl/glfw/v3.3/glfw#Terminate) both state that they can only be called from the main thread.

Unlike the C++ code, we must tell Go not to switch our thread. Add a small init block above the window size constants. This will lock our OS thread and avoid any errors from the GLFW calls.

```golang
func init() {
	runtime.LockOSThread()
}
```

> **Note:** you will need to add the runtime dependency if gofmt doesn't auto add it for you.

You can also research in the [GLFW documentation](https://pkg.go.dev/github.com/go-gl/glfw/v3.3/glfw).

## Keeping the window open

If you run the application at this point, your window will probably close before it even opens.

We will add to our `mainLoop()` to ensure the window stays open until GLFW says the window is closed (or something panics).

```golang
func (app *TriangleApplication) mainLoop() {
	for !app.window.ShouldClose() {
		glfw.PollEvents()
		app.drawFrame()
	}
}
```

[PollEvents](https://pkg.go.dev/github.com/go-gl/glfw/v3.3/glfw#PollEvents) is another method that must be executed by the main thread.

## Conclusion

At this point you have the shell of a vulkan application that lists all the remaining setup steps. You have a window that stays open until the user closes it, and you clean up the GLFW resources.

In relationship to the Vulkan Tutorial, we've completed [Drawing a triangle / Setup / Base code](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Base_code#page_Integrating-GLFW).

The next part will create the Vulkan Instance.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/e5d112d267c6b0bf0dffb607c50783649e8562f3/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/e5d112d267c6b0bf0dffb607c50783649e8562f3#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261))
