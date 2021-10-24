# Vulcan, Go, and A Triangle, Part 1

This tutorial follows my personal execution of the [Vulkan tutorial](https://vulkan-tutorial.com/), with the distinction of being in Go instead of C++.

I started this effort because Go is my preferred programming language and I was interested in understanding more about the modern landscape of GPU programing. While I was able to find a Vulkan tutorial translated for Rust, I could not find an existing one for Go.

While my exploration of Vulkan follows the general approach of the [Vulkan Tutorial](https://vulkan-tutorial.com/), I have done certain steps out of order and try to leverage Go idioms where I can. I also tried to write the code so that most steps start with pseudo-code comments which eventually get expanded into code-blocks.

## Tutorial Scope

This tutorial does not cover the entire vulkan tutorial. Instead it focuses on the [Drawing a Triangle](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Base_code) section of the tutorial.

Since this tutorial is based on the Vulkan Tutorial, the scope is limited in a similar way. It doesn't expect knowledge of OpenGL, but it does expect a basic understanding of computer graphics.

## Approach

The Vulkan Tutorial is more comfortable with changing lines of already written code. The result is that the tutorial adds a concept, then abstracts it when needed.

I generally prefer tutorials to be additive. As a result I generally introduce the abstraction / helper functions earlier in order to avoid changed lines in the diffs.

Often, where the vulkan tutorial adds methods on the application class for each function, I've decided to use anonymous functions inside of a method. I felt this kept many of the related concepts closer together.

Lastly, my implementation panics on failures. I don't really handle them, as this is a tutorial and not a final application.

## Resources

These are some of the resources I used while executing this tutorial.

* The official [Vulkan tutorial](https://vulkan-tutorial.com/) is a great resource, and I'm going to try to avoid plagiarizing from this resources, but given the overlap in our objectives, it is inevitable that I'll overlap with them.
* The [github.com/vulkan-go/vulkan](https://github.com/vulkan-go/vulkan) is what I use for the bridge into all the C code bits. They also provide [Asche](https://github.com/vulkan-go/asche) if you want to skip this tutorial and start using their framework. I would regularly check the [go docs](https://pkg.go.dev/github.com/vulkan-go/vulkan) when I had questions about how C++ signatures were translated.
* You will also need [github.com/go-gl/glfw](https://github.com/go-gl/glfw). There are a couple of points where I referenced the glfw documentation to understand why some things were different in go vs c++. GLFW for Go tends to be more object oriented than the C equivalent (think `window.method(...)` instead of  `function(window, ...)`), making the [go docs](https://pkg.go.dev/github.com/go-gl/glfw/v3.3/glfw) useful for finding a signature for a function.
* You'll also need to install the [Vulkan SDK](https://vulkan.lunarg.com/sdk/home). You can leverage multiple version of vulkan using the python scripts it installs. Remember to use Python3, in case your distro defaults to python2. You may also want to read some guidance on [building MoltenVK](https://github.com/KhronosGroup/MoltenVK#building), should you need it on a Mac.
* You may find the SPIR-V [1.0 spec](https://www.khronos.org/registry/SPIR-V/specs/1.0/SPIRV.pdf) useful at certain points in the tutorial, although I didn't really reference it other than trying to make sure I was reading bytes in the right endian, only to find that it was unnecessary.

## Setting up

Download and install the latest [Vulkan SDK](https://vulkan.lunarg.com/sdk/home) for your platform. This tutorial was created on a Mac, so it was necessary for me to make sure I had installed Molten-VK as part of the SDK. There are also some version available in Homebrew Formulae, but they aren't as up-to-date as what you can download from LunarG.

With all the vulkan stuff installed, create the working directory where you want to do this tutorial, and get all the go bits setup.

```
$ go mod init example.net/vulkan-tutorial
$ go get -u github.com/go-gl/glfw/v3.3/glfw
$ go get -u github.com/vulkan-go/vulkan
```

Lastly run a couple of commands to make sure you can access the vulkan commands for compiling shaders.

```
$ glslc --version
shaderc v2021.2-dev v2021.1-1-g00c8f73
spirv-tools v2021.3-dev v2021.1-48-ge065c482
glslang 11.1.0-203-g0c4c93bf

Target: SPIR-V 1.0
```


## Some helpers that will be useful later.

The following helper functions aren't really part of the vulkan tutorial. They are used during the tutorial in order to keep the tutorial focused on Vulkan code rather than how to solve common programming problems. You are welcome to use your own version if you have them

Only the error handling helper is Vulkan specific. I put all of them in `helpers.go` for simplicity and to keep them out of the tutorial diffs.

### Error Handling

This is technically the first introduction to Vulkan code. Vulkan-go provides a method for converting Vulkan result codes into Go error objects.  If it is passed as "success" result, it returns null. We take advantage of this to eliminate some of error testing in the application and convert them to panics.

For the purposes of this tutorial, we treat most errors as fatal and panic rather than trying to mitigate them.

```golang
package main

import (
	"fmt"
	"bytes"
	"encoding/binary"
	"io/ioutil"

	vk "github.com/vulkan-go/vulkan"
)

// Error handling
func MustSucceed(result vk.Result) {
	err := vk.Error(result)
	if err != nil {
		panic(err)
	}
}
```

If you want to get really fancy, you could copy how [Asche does it](https://github.com/vulkan-go/asche/blob/master/errors.go). For this tutorial, we keep it pretty simple.

### Optional Type

The official tutorial depends on several modern C++ additions to the standard library, including `std::optional`. This type provides a way for us to mimic the optional type in Go.

```golang
// Optional Uint32
type OptionUint32 struct {
	v   uint32
	set bool
}

func (option *OptionUint32) Set(v uint32) {
	option.v = v
	option.set = true
}

func (option OptionUint32) IsSet() bool {
	return option.set
}

func (option OptionUint32) Val() uint32 {
	if !option.IsSet() {
		panic("Attempt to use option value that hasn't been set.")
	}
	return option.v
}
```

### To C Strings.

While I was doing the tutorial, I ran into some problems where strings I passed into vulkan weren't "found" even when I could see them in the enumerated output. I was able to fix the problem by making golang created strings as null terminated. These functions are used to make a string ends with a null.

```golang
// To C Strings
func ToCString(input string) string {
	l := len(input)
	if l == 0 {
		return "\x00"
	} else if input[l-1] != '\x00' {
		return fmt.Sprintf("%s\x00", input)
	}
	return input
}

func ToCStrings(input []string) []string {
	a := make([]string, len(input))
	for k, v := range input {
		a[k] = ToCString(v)
	}
	return a
}
```

### Checking Support

One of the early activities in the vulkan tutorial is checking to see what something supports against what your program requires. This happens with layers, extensions, etc. This set of helpers creates a simple way to convert a slice of strings into a unordered set, and then compare values in a another slice to that set.

It depends on the null terminated versions of the strings as these comparisons are often happening between vulkan responses and golang created requirements.

```golang
// Checking Support
func SliceToMap(keys []string) map[string]bool {
	output := make(map[string]bool)
	for _, v := range keys {
		output[ToCString(v)] = true
	}
	return output
}

func SetSubtraction(a []string, b map[string]bool) []string {
	output := make([]string, 0)
	for _, v := range a {
		if !b[ToCString(v)] {
			output = append(output, v)
		}
	}
	return output
}

func DedupeSlice(a []string) []string {
	idx := make(map[string]bool)
	output := make([]string, 0)
	for _, v := range a {
		if !idx[ToCString(v)] {
			output = append(output, v)
			idx[ToCString(v)] = true
		}
	}
	return output
}

func MustSupport(available, required []string) {
	missing := SetSubtraction(required, SliceToMap(available))
	if len(missing) > 0 {
		err := fmt.Errorf("Required values %v not found in %v.", missing, available)
		panic(err)
	}
}
```

### Clamp-able

Golang doesn't come with a built-in max or min, so we create a couple of helper functions to enable clamping values. This becomes more useful when selecting a value between ranges supported by the hardware.

```golang
// Clamp-able
func ClampUint32(v, smallest, largest uint32) uint32 {
	return MaxUint32(smallest, MinUint32(v, largest))
}

func MaxUint32(x, y uint32) uint32 {
	if x < y {
		return y
	}
	return x
}

func MinUint32(x, y uint32) uint32 {
	if x > y {
		return y
	}
	return x
}
```

### 32-bit Words

When we start working with shaders, the byte code passed in as `[]uint32`. This type provides simple way to load 32bit words, without having to focus on the IO boilerplate in the main tutorial. I've hard coded the LittleEndian byte order for simplicity, because my development machine is little-endian.

```golang
// Must read a file
func MustReadFile(fn string) []byte {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		panic(err)
	}
	return b
}

// 32-bit Words
type WordsUint32 []uint32

func NewWordsUint32(b []byte) WordsUint32 {
	r := bytes.NewReader(b)
	words := make([]uint32, len(b)/4)
	binary.Read(r, binary.LittleEndian, words)
	return WordsUint32(words)
}

func (words WordsUint32) Sizeof() uint {
	return uint(len(words) * 4)
}
```

## Adding the skeleton.

To end this chapter in a buildable state, we add the basic skeleton of the application into `main.go`.

```golang
package main

type TriangleApplication struct{}

func (app *TriangleApplication) setup() {}

func (app *TriangleApplication) mainLoop() {}

func (app *TriangleApplication) drawFrame() {}

func (app *TriangleApplication) recreatePipeline() {}

func (app *TriangleApplication) cleanup() {}

func (app *TriangleApplication) Run() {
	app.setup()
	defer app.cleanup()
	app.mainLoop()
}

func main() {
	app := TriangleApplication{}
	app.Run()
}
```

In `main()`, we create an instance of our application and invoke `Run()`. As the tutorial progresses, we will add some application specific fields and constants into the Application structure.

In `Run()`, we invoke `setup()`, defer an invocation of `cleanup()`, and then start the `mainloop()`.

Eventually, `setup()` is going to do all of the application specific setup. This will include creating the window, selecting devices, and building the pipeline. `cleanup()` will do the reverse by releasing and destroying the objects created during setup.

When complete, `mainloop()` deal with events from GLFW, call `drawFrame()` and `recreatePipeline()` as needed.

## Conclusion

This part covered the introduction to the tutorial, provided some resources to help throughout the tutorial, creating some helpers that will be used through out the rest of the tutorial, and a basic application skeleton.

In relation to the Vulkan Tutorial, We are at [Integrating GLFW](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Base_code#page_Integrating-GLFW).

The next part will provide the skeleton used by `Setup()`.

[main.go](https://github.com/ibd1279/vulkangotutorial/blob/49963f40e98395fdfee9661702008bb2a9503d81/main.go) / [helpers.go](https://github.com/ibd1279/vulkangotutorial/blob/49963f40e98395fdfee9661702008bb2a9503d81/helpers.go)
