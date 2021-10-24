# vulkangotriangle

This tutorial follows my personal execution of the [Vulkan tutorial](https://vulkan-tutorial.com/), with the distinction of being in Go instead of C++. The steps of each commit are available on [Vulkan, Go, and a Triangle](https://virtsoftcrazy.blogspot.com/2021/10/vulcan-go-and-triangle-part-1.html) and in the repository under the [tutorial folder](https://github.com/ibd1279/vulkangotutorial/tree/main/tutorial).

I started this effort because Go is my preferred programming language and I was interested in understanding more about the modern landscape of GPU programing. While I was able to find a Vulkan tutorial translated for Rust, I could not find an existing one for Go.

While my exploration of Vulkan follows the general approach of the [Vulkan Tutorial](https://vulkan-tutorial.com/), I have done certain steps out of order and try to leverage Go idioms where I can. I also tried to write the code so that most steps start with pseudo-code comments which eventually get expanded into code-blocks.

## Tutorial Scope

This tutorial does not cover the entire vulkan tutorial. Instead it focuses on the [Drawing a Triangle](https://vulkan-tutorial.com/Drawing_a_triangle/Setup/Base_code) section of the tutorial.

Since this tutorial is based on the Vulkan Tutorial, the scope is limited in a similar way. It doesn't expect knowledge of OpenGL, but it does expect a basic understanding of computer graphics.

## Resources

These are some of the resources I used while executing this tutorial.

* The official [Vulkan tutorial](https://vulkan-tutorial.com/) is a great resource, and I'm going to try to avoid plagiarizing from this resources, but given the overlap in our objectives, it is inevitable that I'll overlap with them.
* The [github.com/vulkan-go/vulkan](https://github.com/vulkan-go/vulkan) is what I use for the bridge into all the C code bits. They also provide [Asche](https://github.com/vulkan-go/asche) if you want to skip this tutorial and start using their framework. I would regularly check the [go docs](https://pkg.go.dev/github.com/vulkan-go/vulkan) when I had questions about how C++ signatures were translated.
* You will also need [github.com/go-gl/glfw](https://github.com/go-gl/glfw). There are a couple of points where I referenced the glfw documentation to understand why some things were different in go vs c++. GLFW for Go tends to be more object oriented than the C equivalent (think `window.method(...)` instead of  `function(window, ...)`), making the [go docs](https://pkg.go.dev/github.com/go-gl/glfw/v3.3/glfw) useful for finding a signature for a function.
* You'll also need to install the [Vulkan SDK](https://vulkan.lunarg.com/sdk/home). You can leverage multiple version of vulkan using the python scripts it installs. Remember to use Python3, in case your distro defaults to python2. You may also want to read some guidance on [building MoltenVK](https://github.com/KhronosGroup/MoltenVK#building), should you need it on a Mac.
* You may find the SPIR-V [1.0 spec](https://www.khronos.org/registry/SPIR-V/specs/1.0/SPIRV.pdf) useful at certain points in the tutorial, although I didn't really reference it other than trying to make sure I was reading bytes in the right endian, only to find that it was unnecessary.