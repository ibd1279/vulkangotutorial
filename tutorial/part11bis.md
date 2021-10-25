# Vulcan, Go, and A Triangle, Part 11 bis

In part 10, I mentioned trying to use go routines to create the command buffers, but that didn't work because the command pool and the command buffers must all be operated on in a single thread.

> I had tried doing the command recording in parallel using go-routines, but that resulted validation layer threading errors. A command pool is apparently thread specific.

The rest of this part explores using a locked thread go routine associated with a command pool in order to multi-thread command recording.

**Note** This is basically building an abstraction layer on top of Vulkan, and isn't necessary to understand how vulkan works. As this code is a wholesale divergence from the Vulkan tutorial, I won't be using it when I start on the next section.

<!--more-->

## Some background

Now that I had a working application, I decided to read a bit more on how I can parallelize work with the command pools. According to the [Vulkan 1.2 specification](https://www.khronos.org/registry/vulkan/specs/1.2-extensions/man/html/VkCommandPool.html),

> Command pools are externally synchronized, meaning that a command pool must not be used concurrently in multiple threads. That includes use via recording commands on any command buffers allocated from the pool, as well as operations that allocate, free, and reset command buffers or the pool itself.

The same is also apparently true for DescriptorPools, which I'll be using later. I found this [Arm community posting](https://community.arm.com/arm-community-blogs/b/graphics-gaming-and-vr-blog/posts/vulkan-mobile-best-practices-and-management) to be particularly helpful when doing some background research.

Go hides a lot of the threading logic behind the scenes and provides little to no language level constructs for looking it up, as this is an [intentional design of the language](https://golang.org/doc/faq#no_goroutine_id).

That means I need to create a mechanism for locking a CommandPool to a specific thread, and the find a way to farm work into that thread.

## Channels

The go-centric solution to this problem is channels. Spin up a new go-routine for doing work and use channels to do communication and memory sharing.  The second part of the solution is `	runtime.LockOSThread()`, which was introduced in part 2 for GLFW.

Rather than creating a command pool directly in the main thread, I'd start a new go-routine to lock the thread, create the command pool and loop pulling recording work from a channel. This is philosophically similar to how Queues work in Vulkan.

## Three Pieces

I broke the problem down into three parts: creating the dedicated thread, making the thread consume work, and finally making the thread exit.

Creating the dedicated thread was the easiest, so I started there.

### New Thread

Even before I had defined the associated type, I created a new function to start the go-routine, lock the thread, and create the CommandPool.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		
		var cmdPool vk.CommandPool
		ret := vk.CreateCommandPool(
			device,
			createInfo,
			allocator,
			&cmdPool,
		)
		if ret == vk.Success {
			vk.DestroyCommandPool(device, cmdPool, allocator)
		}
	}()
}
```

But then I realized that I needed a mechanism to put the `CreateCommandPool` result back into the main thread and block until the CommandPool is ready.

Channels again to the rescue.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	createResultChan := make(chan vk.Result, 0)
	defer close(createResultChan)
	go func() {
		...
		createResultChan <- ret
		if ret == vk.Success {
			vk.DestroyCommandPool(device, cmdPool, allocator)
		}
	}()
	
	return <-createResultChan
}
```

Now the NewCommandPoolThread function won't return until we have the result of the `CreateCommandPool` call.

### Using Channels as a queue

Now that I had a CommandPool in a dedicated thread, I needed to be able to submit CommandBuffer allocation and free requests. I decided to introduce a request and response type, as I needed to pass different parameters groups together into a channel.

```golang
type CommandPoolThreadRequest struct {
	backChan  chan<- CommandPoolThreadResponse
}

type CommandPoolThreadResponse struct {}
```

The backChan is basically a Vulkan fence, but in Go; it is used to allow coordination with the caller.

The second type is for returning result of the request.

I used these two types to introduce a channel for submitting new work.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	requestChan := make(chan CommandPoolThreadRequest, 0)
	
	...
}
```

Then I needed to add a loop into the go-routine to consume requests on this channel.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	...
	go func() {
		...
				
		var loop bool
		if ret == vk.Success {
			loop = true
		}
		
		for loop {
			select {
			case request := <-requestChan:
				// Do something.
				
				// Send the response back.
				request.backChan <- CommandPoolThreadResponse{}
				close(request.backChan)
			}
		}
		if ret == vk.Success {
			vk.DestroyCommandPool(device, cmdPool, allocator)
		}
	}()
	...
}
```

I also needed to close the channel when I was done with it.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	...
	go func() {
		defer close(requestChan)
		...
	}()
	...
}
```

Now I have a mechanism for consuming work, but no mechanism for ending the go-routine once it has started.

### Making the thread exit

Similar to submitting work, ending the thread is another channel to signal exiting the loop.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	signalFinishedChan := make(chan chan<- struct{}, 0)
	...
}
```

Then I needed a case in the select to capture this single and flip the value of `loop`.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	...
	go func() {
		...
		var finishedChan chan<- struct{}
		for loop {
			select {
			case finishedChan = <-signalFinishedChan:
				loop = false
			case request := <-requestChan:
				...
			}
		}
		...
		if finishedChan != nil {
			finishedChan <- struct{}{}
			close(finishedChan)
		}
	}()
	...
}
```

I also needed to added the close to the singalFinishedChan.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	...
	go func() {
		defer close(signalFinishedChan)
		...
	}()
	...
}
```

### Making it usable.

I can almost start and end this thread. The thing missing is returning all these channels in a way that makes them useable.

```golang
type CommandPoolThread struct {
	sfChan chan<- chan<- struct{}
	rChan  chan<- CommandPoolThreadRequest
}
```

Then I updated the signature on the `NewCommandPoolThread` to return this type in addition to the `vk.Result`. The return statement also needed to be updated.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  (*CommandPoolThread, vk.Result) {
	...
	ret := <-createResultChan
	var cpt *CommandPoolThread
	if ret == vk.Success {
		cpt = &CommandPoolThread{
			sfChan: signalFinishedChan,
			rChan: requestChan,
		}
	}
	return cpt, ret
}
```

I added two methods to the new type. One for adding work, and one for destroying the thread. 

```golang
func (cpt *CommandPoolThread) Destroy() <-chan struct{} {
	c := make(chan struct{}, 0)
	cpt.sfChan <- c
	return c
}

func (cpt *CommandPoolThread) Submit(req CommandPoolThreadRequest) <-chan CommandPoolThreadResponse {
	c := make(chan CommandPoolThreadResponse, 1)
	req.backChan = c
	cpt.rChan <- req
	return c
}
```

These are wrappers for creating the required coordination channel, and pushing work into the queue.

## Do something Vulkan

In order to do something Vulkan, I needed to expand the request and response types.

```golang
type CommandPoolThreadRequest struct {
	backChan  chan<- CommandPoolThreadResponse
	Alloc     vk.CommandBufferAllocateInfo
	Buffers   []vk.CommandBuffer
	Commands  func(int, vk.CommandBuffer) error
}

type CommandPoolThreadResponse struct {
	Alloc    vk.Result
	Commands []error
	Buffers  []vk.CommandBuffer
}
```

I'm overloading all situations into a single Request, which isn't good design but works for this experiment.

Then I expanded the `// Do something.` comment in the for loop.

```golang
func NewCommandPoolThread(device vk.Device, createInfo *vk.CommandPoolCreateInfo, allocator *vk.AllocationCallbacks)  vk.Result {
	...
	go func() {
		...
		var finishedChan chan<- struct{}
		for loop {
			select {
			...
			case request := <-requestChan:
				// Do something.
				// Free buffers in the request.
				if len(request.Buffers) > 0 {
					vk.FreeCommandBuffers(
						device,
						cmdPool,
						uint32(len(request.Buffers)),
						request.Buffers,
					)
				}
				
				// Allocate new buffers.
				if request.Alloc.CommandBufferCount > 0 {					
					response := CommandPoolThreadResponse{}
					response.Buffers = make(
						[]vk.CommandBuffer,
						request.Alloc.CommandBufferCount,
					)
					
					request.Alloc.CommandPool = cmdPool
					response.Alloc = vk.AllocateCommandBuffers(
						device,
						&request.Alloc,
						response.Buffers,
					)
					if response.Alloc != vk.Success {
						request.backChan <- response
					} else {
						response.Commands = make([]error, len(response.Buffers))
						for k, cmdBuffer := range response.Buffers {
							response.Commands[k] = request.Commands(k, cmdBuffer)
						}
						request.backChan <- response
					}
				} 
				
				// Send the response back.
				request.backChan <- CommandPoolThreadResponse{}
				close(request.backChan)
			}
		}
		...
	}()
	...
}
```

If buffers are provided in the request, they are freed. If buffers are to be allocated, they are allocated and then Commands is called on each allocated CommandBuffer.

## Conclusion

As an experiment, I did retrofit this into the Vulkan Tutorial code to test it. While everything did work, The Vulkan Tutorial application isn't complex enough for this. Since the CommandBuffers are the last thing I create in the pipeline, I had to block immediately after submitting the buffer creation.

In reality, this whole concept would need to be expanded to support more than just a command buffer. You may want multiple command buffers on a single thread. Or a CommandBuffer and a DescriptorPool linked together because they work with the same data.

The [code as a gist](https://gist.github.com/ibd1279/ba12f012b20a839f8aca3afa0cfe64bc).
