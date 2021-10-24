# Vulcan, Go, and A Triangle, Part 4

In this part of the tutorial, I'm going to inspect what extensions and layers are available for an instance. The call to `vk.CreateInstance` can result in `vk.ErrorLayerNotPresent` or `vk.ErrorExtensionNotPresent` according to [the Vulkan spec](https://www.khronos.org/registry/vulkan/specs/1.2-extensions/man/html/vkCreateInstance.html). By inspecting the available options and checking if my required options are supported, I can provide a more debuggable error response.

Following the Vulkan Tutorial, I implemented the necessary functions to enumerate over available layers and extensions before calling CreateInstance. This part relates to [Drawing a triangle / Setup / Instance / Checking for extension support](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Instance#page_Checking-for-extension-support) in the original tutorial.

<!--more-->

## The other common pattern

As mentioned in part 3, there are two common patterns while working with the vulkan APIs. In order to enumerate over vulkan data, we will be exploring the pattern of calling a function twice: once for getting the count and a second time for getting the data.

In pseudo-code, it looks something like this:

```golang
func EnumerateSomething() []vk.Something {
	// Allocate the count.
	
	// Call to get the count.
	
	// Allocate to store the data.
	
	// Call to get the data.
	
	// Return the result.
}
```

## Dealing with Deref()

While doing the tutorial, I ran into a couple of places where I had forgotten to call `Deref()` on objects. After wasting a bunch of time trying to figure out why I had zero memory or zero sized extents, I decided to add some wrapper functions to explicitly call `Deref()` for me.

I started by adding a function to get the instance [LayerProperties](https://pkg.go.dev/github.com/vulkan-go/vulkan#LayerProperties) at the end of the main file.

```golang
// LayerProperties
func EnumerateInstanceLayerProperties() []vk.LayerProperties {
	// Allocate the count.
	
	// Call to get the count.
	
	// Allocate to store the data.
	
	// Call to get the data.
	
	// Dereference the data.
	
	// Return the result.
}
```

The first step was to create a variable to hold the count, and call the function to populate it.

```golang
// LayerProperties
func EnumerateInstanceLayerProperties() []vk.LayerProperties {
	// Allocate the count.
	var count uint32
	
	// Call to get the count.
	vk.EnumerateInstanceLayerProperties(&count, nil)
	...
}
```

Many of the enumeration functions in Vulkan take a pointer to the count and accept nil for the data. This allows for fetching the count without copying or allocating any data in the application.

Once I had the count, I allocated storage for the result and called a second time.

```golang
// LayerProperties
func EnumerateInstanceLayerProperties() []vk.LayerProperties {
	...
	// Allocate to store the data.
	list := make([]vk.LayerProperties, count)
	
	// Call to get the data.
	vk.EnumerateInstanceLayerProperties(&count, list)
	...
}
```

Now that I have a collection of vulkan objects, we need to call `Deref()` on each in order pull the data into the Go and return the result.

```golang
// LayerProperties
func EnumerateInstanceLayerProperties() []vk.LayerProperties {
	...
	// Dereference the data.
	for k, _ := range list {
		list[k].Deref()
	}

	// Return the result.
	return list
}
```

This pattern appears often. In fact, I implemented a second version of it immediately after this one for the Extension Properties.

```golang
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
```

## Properties into something printable.

Now that I can load data about layers and extensions from the Vulkan instance, we create two methods to convert the properties objects into usable strings.

```golang
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
```

The main thing here is converting Vulkan strings into Go strings using the [vk.ToString](https://pkg.go.dev/github.com/vulkan-go/vulkan#ToString) function.

## List what is supported.

I used these new functions at the top of the createInstance function in order to output what is supported by the Vulkan instance.

```golang
func (app *TriangleApplication) setup() {
	...
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

		// Available Instance Extensions.
		layerExts := EnumerateInstanceExtensionProperties("")
		availExtNames := ExtensionPropertiesNames(layerExts)
		for h := 0; h < len(layerExts); h++ {
			fmt.Printf("Extension Avail: %s\n",
				availExtNames[h])
		}
		
		// Required Instance Extensions.
		
		// Create the info object.
		...
	}	
	...
}
```

> Note: My editor auto imported "fmt". So I didn't include it as part of adding this output.

At this point I ran the application to verify that I saw some output. The layer descriptions are pretty anemic (especially since they are practically the same words as the name).

I admit that I was happy to start seeing something Vulkan output, even if most of the code written for this part wasn't very Vulkan specific. 

## Requiring something

Now that I had a way to see what is supported, I was ready to start adding some required extensions and layers. Specifically, the validation layers (VK_LAYER_KHRONOS_validation) and whatever  extensions GLFW requires (`window.GetRequiredInstanceExtensions()`).

I started by adding two new fields onto the application for specifying which layers and extensions I wanted to explicitly require:

```golang
type TriangleApplication struct {
	window                         *glfw.Window
	instance                       vk.Instance
	RequiredInstanceExtensionNames []string
	RequiredInstanceLayerNames     []string
}
```

Then I added the values to the initialization in the main function.

```golang
func main() {
	app := TriangleApplication{
		RequiredInstanceExtensionNames: []string{},
		RequiredInstanceLayerNames: []string{
			"VK_LAYER_KHRONOS_validation",
		},
	}
	app.Run()
}
```

I added the required GLFW extensions at the end of the createWindow function.

```golang
func (app *TriangleApplication) setup() {
	// Steps.
	createWindow := func() {
		...
		// Update required extensions.
		app.RequiredInstanceExtensionNames = append(
			app.RequiredInstanceExtensionNames,
			app.window.GetRequiredInstanceExtensions()...,
		)
	}
	...
}
```

I'm blindly appending the GLFW result to the required extensions because I expect to do the deduplication in the CreateInstance function.

The createInstance function can now be updated to enforce the requirements before calling vk.CreateInstance.

```golang
func (app *TriangleApplication) setup() {
	...
	createInstance := func() {
		...
		// Required Instance Layers.
		reqLayerNames := ToCStrings(DedupeSlice(app.RequiredInstanceLayerNames))
		MustSupport(availLayerNames, reqLayerNames)
		...
		// Required Instance Extensions.
		reqExtNames := ToCStrings(DedupeSlice(app.RequiredInstanceExtensionNames))
		MustSupport(availExtNames, reqExtNames)
		...
	}	
	...
}
```

> **Note**: My helper functions normalize everything into a null terminated string to ensure our Go versions match with the Vulkan C-Style versions. I took that step because the null terminated aspect of the Vulkan C strings wasted a lot of time in debugging before I realized that was the difference.

I admit to having played around with some non existing layer and extension names to make sure the MustSupport functions were working as expected.

Then I updated the instance create info to use the new required names.

```golang
func (app *TriangleApplication) setup() {
	...
	createInstance := func() {
		...
		// Create the info object.
		instanceInfo := vk.InstanceCreateInfo{
			...
			EnabledExtensionCount:   uint32(len(reqExtNames)),
			PpEnabledExtensionNames: reqExtNames,
			EnabledLayerCount:       uint32(len(reqLayerNames)),
			PpEnabledLayerNames:     reqLayerNames,
		}
		...
	}	
	...
}
```

At this point I had created a Vulkan instance should work with my window and would warn me when I was doing things incorrectly.

## Conclusion

This part introduced the second common vulkan API pattern, and leveraged many of our helper functions to provide debug functionality. We also enabled the Validation layer for our application and ensured that our vulkan instance supports our GLFW window.

In the next part we will focus on enumerating and selecting a physical device.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/9693a0d47a04b2eca069955c45a9206f1d123ac3/main.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/9693a0d47a04b2eca069955c45a9206f1d123ac3#diff-2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261))
