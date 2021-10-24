# Vulcan, Go, and A Triangle, Part 3

In this part of the tutorial, we are going to initialize a vulkan instance. The vulkan instance is the connection between your application and the Vulkan framework. It allows the application to enumerate physical devices and supported functionality.

This part relates to [Drawing a triangle / Setup / Instance](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Instance#page_Cleaning-up) in the original tutorial.

<!--more-->

## Integrating Vulkan

First, add the necessary import to start using the vulkan package.

```golang
import (
	"runtime"
	
	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)
```

I aliased the package to vk. This is recommended by the vulkan-go developers and makes most of the function names look similar to their C counterparts.


Before we can start calling vulkan functions, we need to connect GLFW and Vulkan together, and call `vk.Init()`. Update the `createInstance` to be the following:

```golang
func (app *TriangleApplication) setup() {
	...
	
	initVulkan := func() {
		// Link Vulkan and GLFW
		vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())

		// Initialize Vulkan
		if err := vk.Init(); err != nil {
			panic(err)
		}
	}
	
	...
}
```

This isn't covered in the Vulkan tutorial, but this is required for setting up the pointers and addresses that are necessary for Vulkan-go. I separated it from the instance creation so that createInstance could remain similar in content to the [Vulkan Tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Instance#page_Creating-an-instance) version.

## Creating an Instance

Add the vulkan instance handle to the application.

```golang
type TriangleApplication struct {
	window   *glfw.Window
	instance vk.Instance
}
```

The instance handle will be used to request information about capabilities, layers, and physical devices.

### A common pattern

There are two common patterns you will encounter while working with vulkan. The first is populating an info structure before invoking a creation or allocation function. The second is calling an enumeration function twice for count and lists.

We will be using the first pattern as part of creating our instance: create the info structure, create the result object, and call the Vulkan function. We start by populating the createInstance function with pseudo-code for the pattern.

```golang
func (app *TriangleApplication) setup() {
	...
	
	createInstance := func() {
		// Create the info object.

		// Create the result object.

		// Call the Vulkan function.

		// Update the application.
	}
	
	...
}
```

The first step is to populate a structure with values. [InstanceCreateInfo](https://pkg.go.dev/github.com/vulkan-go/vulkan#InstanceCreateInfo) is a good prototype for many of the structures. They usually start with an SType field which much match the type of the structure. There is a [good answer on stack overflow](https://stackoverflow.com/questions/36347236/vulkan-what-is-the-point-of-stype-in-vkcreateinfo-structs) if you are curious as to why.

They also include a PNext for future functionality. Fields that start with a P are for pointers, nested structures, or lists. The info objects can get pretty large, like the [GraphicsPipelineCreateInfo](https://pkg.go.dev/github.com/vulkan-go/vulkan#GraphicsPipelineCreateInfo) for example.

### Create the info object

The info object is mostly copied straight from the Vulkan Tutorial.

```golang
func (app *TriangleApplication) setup() {
	...
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
		...
	}	
	...
}
```

For the embedded `vk.ApplicationInfo` structure, it is important to remember that Vulkan expects null terminated strings, so it is necessary to use the `ToCString()` helper function that we created in Part 1 when passing a string to Vulkan. Most of the structure is dedicated to telling Vulkan the identity of your application.

For this step in the tutorial, we could have ignored layers and extensions fields because Go would have initialized the counts to `0` by default. We added them with default values as they become important later in this tutorial, and to illustrate the common aspects of Vulkan APIs around Count-suffixed fields and P-prefixed fields.

## Call the Vulkan function

The `vk.CreateInstance()` call is pretty straightforward, so I jumped directly into calling it.

```golang
func (app *TriangleApplication) setup() {
	...
	createInstance := func() {
		...
		
		// Call the Vulkan function.
		MustSucceed(vk.CreateInstance(&instanceInfo, nil, &app.instance))
		
		...
	}
	...
}
```

At this point in the code will build, but it will panic when run. I got the follow error message when running it.

```
panic: runtime error: cgo argument has Go pointer to Go pointer
```

There is an [open issue on Vulkan-go](https://github.com/vulkan-go/vulkan/issues/42) for this topic.

### Create the result object and Update

The easiest work around I was able to define was using a local variable to capture the result of calls. 

```golang
func (app *TriangleApplication) setup() {
	...
	createInstance := func() {
		...
		
		// Create the result object.
		var instance vk.Instance
		
		// Call the Vulkan function.
		MustSucceed(vk.CreateInstance(&instanceInfo, nil, &instance))
		
		// Update the application.
		app.instance = instance
	}
	...
}
```

If you build and run the application now, it won't panic.

### One more thing

For this tutorial, we add a call `vk.InitInstance()` to avoid a seg-fault later. [According to the documentation](https://pkg.go.dev/github.com/vulkan-go/vulkan#InitInstance), this is required for macOS, but it shouldn't have a negative impact for other platforms.

```golang
func (app *TriangleApplication) setup() {
	...
	createInstance := func() {
		...
		
		// InitInstance is required for macOs?
		vk.InitInstance(app.instance)		
	}
	...
}
```


## Cleanup

We need to tell Vulkan when we are done with the instance. Add a call to `vk.DestroyInstance` to the top of the `TriangleApplication#cleanup` method.

```golang
func (app *TriangleApplication) cleanup() {
	vk.DestroyInstance(app.instance, nil)
	app.window.Destroy()
	glfw.Terminate()
}
```

Some vulkan functions, especially those in the create and destroy family, take an optional pointer to the allocation callback. This tutorial doesn't do anything special with memory management, so we pass nil for the allocator to ignore that functionality.

## Conclusion

This part focused on the pattern most common to creating new vulkan objects. It also provided a high level overview of the fields in info structures. These two concepts will be encountered often in the following parts.

Now that I have the instance handle, I can start enumerating what that instance supports and if it will meet the needs of my application. That will be the focus in the next part.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/1a56f54f0d3561f6c0042ae80d8c7ad05544e95e/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/1a56f54f0d3561f6c0042ae80d8c7ad05544e95e#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261))
