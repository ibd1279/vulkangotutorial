# Vulcan, Go, and A Triangle, Part 9

In this part, I made the pipeline objects. This includes loading the shader modules, configuring the fixed functions, and creating the Vulkan pipelines. This part started with [Drawing a triangle / Graphics pipeline basics / Shader modules](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Shader_modules), jumps into the [Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions), and ends with the [Conclusion](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Conclusion).

<!--more-->

## Create and compile the shaders

Nothing about this step was related to Go, so I basically followed the instructions from the Vulkan Tutorial. I Started with the [Vertex shader](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Shader_modules#page_Vertex-shader) and continued until I had completed [Compiling the shaders](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Shader_modules#page_Compiling-the-shaders).

The TL;DR version was to put the code for the two shaders into a directory named "shaders". Then, use the `glslc` command we tested in part 1 to compile them.

```shell
$ glslc shader.vert -o vert.spv
$ glslc shader.frag -o frag.spv
```

This Resulted in having two `.spv` files created. I also added that file extension to [.gitignore](https://github.com/ibd1279/vulkangotutorial/blob/14e833ffe2d9bf32d11ec69aeb3203f4c858f06b/.gitignore#L7), because I didn't want to commit the bytecode on accident.

## The basic skeleton

Creating the pipeline was closer to creating the swapchain than it was to creating the pipeline layout: there were many objects that needed to be setup before I could populate the create info object.

I started by adding the pipelines field to the Pipeline type.

```golang
type Pipeline struct {
	...
	Pipelines      []vk.Pipeline
}
```

The most interesting thing here is that I returned a slice of pipelines instead of a single one. While I will only be implementing a single pipeline, the Vulkan APIs for [CreateGraphicsPipelines](https://pkg.go.dev/github.com/vulkan-go/vulkan#CreateGraphicsPipelines) and [CreateComputePipelines](https://pkg.go.dev/github.com/vulkan-go/vulkan#CreateComputePipelines) are designed to work in plurals, so I copied that into my implementation.

Then I added the skeleton of the pipeline creation.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the pipelines.
	pipelines := func() []vk.Pipeline {
		// Function for loading a shader.
	
		// Create the vertex shader.

		// Create the fragment shader.

		// Create the ShaderStage info objects.

		// Create the info object.

		// Create the result object.

		// Call the Vulkan function.

		// Return the pipelines.
		return nil
	}()

	// Create and return the pipeline
	return &Pipeline{
		Swapchain:             swapchain,
		SwapchainImages:       swapchainImages,
		SwapchainImageFormat:  format.Format,
		SwapchainExtent:       extent,
		SwapchainImageViews:   imageViews,
		SwapchainFramebuffers: framebuffers,
		RenderPass:            renderPass,
		PipelineLayout:        pipelineLayout,
		Pipelines:             pipelines,
	}
}
```

The function returned nil during this phase because I wanted to compile and run before the function was complete. Mostly to make sure the shaders properly loaded.

## Loading a shader

In part 1, I created two helper functions to simplify the loading of shader bytecode: `MustReadFile()` and `NewWordsUint32()`.

In this part, I used those two functions to create the Vulkan Shader module. The vulkan part of the code is the usual create pattern.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	// Create the pipelines.
	pipelines := func() []vk.Pipeline {
	  // Function for loading a shader.
		loadShaderModule := func (fn string) vk.ShaderModule {
			// load the shader bytes
			shaderWords := NewWordsUint32(MustReadFile(fn))
		
			// Create the info object.
			shaderInfo := vk.ShaderModuleCreateInfo{
				SType:    vk.StructureTypeShaderModuleCreateInfo,
				CodeSize: shaderWords.Sizeof(),
				PCode:    []uint32(shaderWords),
			}
			
			// Create the result object.
			var shaderModule vk.ShaderModule
		
			// Call the Vulkan function.
			MustSucceed(vk.CreateShaderModule(app.device, &shaderInfo, nil, &shaderModule))
		
			// return the handle
			return shaderModule
		}
		
		...
	}()
	...
}
```

WordsUint32 is effectively a Sizeof() function added to a `[]uint32`. I created it to make creating the info object easier and to avoid needing to use `unsafe` to do the slice conversion.

## Creating the shader stage

Creating the vertex and fragment shader was simple, as I already created a function to load them into shader modules.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the vertex shader
		vertShaderModule := loadShaderModule("shaders/vert.spv")
		defer vk.DestroyShaderModule(app.device, vertShaderModule, nil)

		// Create the fragment shader
		fragShaderModule := loadShaderModule("shaders/frag.spv")
		defer vk.DestroyShaderModule(app.device, fragShaderModule, nil)

		...
	}()
}
```

The shader module is only needed for the CreateGraphicsPipeline call, so I used `defer` to clean it up when this function ends.

The shader modules needed to be put into shader stage info objects so that I could use them in the pipeline create info.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the ShaderStage info objects.
		shaderStages := []vk.PipelineShaderStageCreateInfo{
			vk.PipelineShaderStageCreateInfo{
				SType:  vk.StructureTypePipelineShaderStageCreateInfo,
				Stage:  vk.ShaderStageVertexBit,
				Module: vertShaderModule,
				PName:  ToCString("main"),
			},
			vk.PipelineShaderStageCreateInfo{
				SType:  vk.StructureTypePipelineShaderStageCreateInfo,
				Stage:  vk.ShaderStageFragmentBit,
				Module: fragShaderModule,
				PName:  ToCString("main"),
			},
		}

		...
	}()
}
```

These calls all described how the pipeline was going to use a particular shader module. For more information on the fields (and the optional `PSpecializationInfo` that I didn't specify), see the [Vulkan Tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Shader_modules#page_Shader-stage-creation).

## Fixed functions

Rather than creating a bunch of independent info objects, I declared all of them inline for the RAII. As a result, I found it easier to work backwards from the [Conclusion](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Conclusion) fields and find the fixed function declaration as needed.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				SType:               vk.StructureTypeGraphicsPipelineCreateInfo,
				StageCount:          uint32(len(shaderStages)),
				PStages:             shaderStages,
				PVertexInputState:   &vk.PipelineVertexInputStateCreateInfo{},
				PInputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{},
				PViewportState:      &vk.PipelineViewportStateCreateInfo{},
				PRasterizationState: &vk.PipelineRasterizationStateCreateInfo{},
				PMultisampleState:   &vk.PipelineMultisampleStateCreateInfo{},
				PColorBlendState:    &vk.PipelineColorBlendStateCreateInfo{},
				Layout:              pipelineLayout,
				RenderPass:          renderPass,
				Subpass:             0,
			},
		}
		...
	}()
	...
}
```

Each one of these fixed function pointers can be found in [Drawing a triangle / Graphics pipeline basics / Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions).

As there is very little Go-specific code involved in this, I recommend following the Vulkan Tutorial to understand the "why". Links to the relevant sections are listed after the code.

### Vertex input 

The vertex data was already in the shaders, but I needed to tell the pipeline that we won't be providing anything.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				...
				PVertexInputState: &vk.PipelineVertexInputStateCreateInfo{
					SType:                           vk.StructureTypePipelineVertexInputStateCreateInfo,
					VertexBindingDescriptionCount:   0,
					VertexAttributeDescriptionCount: 0,
				},
				...
			},
		}
		...
	}()
	...
}
```

Details are available in the [Vertex input part of the Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Vertex-input) section on the Vulkan Tutorial.

### Input assembly

Input assembly is explaining the type of geometry I will be drawing.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				...
				PInputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
					SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
					Topology:               vk.PrimitiveTopologyTriangleList,
					PrimitiveRestartEnable: vk.False,
				},
				...
			},
		}
		...
	}()
	...
}
```

Details are available in the [Input assembly part of the Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Input-assembly).

### Viewport

The viewport and scissor describe how to render the output to the frame buffer. I found the explanatory graphic provided in the [Vulkan tutorial](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Viewports-and-scissors) very helpful.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				...
				PViewportState: &vk.PipelineViewportStateCreateInfo{
					SType:         vk.StructureTypePipelineViewportStateCreateInfo,
					ViewportCount: 1,
					PViewports: []vk.Viewport{
						vk.Viewport{
							Width:    float32(extent.Width),
							Height:   float32(extent.Height),
							MaxDepth: 1.0,
						},
					},
					ScissorCount: 1,
					PScissors: []vk.Rect2D{
						vk.Rect2D{
							Offset: vk.Offset2D{},
							Extent: extent,
						},
					},
				},
				...
			},
		}
		...
	}()
	...
}
```

The origin values (x, y) default to zero in go, so I left them out to reduce the size of the structure. As stated in the [Viewport and Scissors part of the Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Viewports-and-scissors), I'm using the full size of the swapchain extent.

There are also some helpful comments at the bottom of the Vulkan Tutorial for configuring Viewport and Triangles as [Dynamic state](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Dynamic-state). I skipped that for now, as the tutorial mentions it will get to dynamic state in later chapters.

### Rasterization

The vertex information describes the image in terms of vectors, but the screen displays pixels. The rasterization stage converts between the two.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				...
				PRasterizationState: &vk.PipelineRasterizationStateCreateInfo{
					SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
					DepthClampEnable:        vk.False,
					RasterizerDiscardEnable: vk.False,
					PolygonMode:             vk.PolygonModeFill,
					LineWidth:               1.0,
					CullMode:                vk.CullModeFlags(vk.CullModeBackBit),
					FrontFace:               vk.FrontFaceClockwise,
					DepthBiasEnable:         vk.False,
				},
				...
			},
		}
		...
	}()
	...
}
```

Details are available in the [Rasterizer part of the Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Rasterizer).

### Multisample

Multisampling is a solution to antialiasing. These settings disabled it. The tutorial has a much later chapter dedicated to multisampling, when I assume it will return to this topic.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				...
				PMultisampleState: &vk.PipelineMultisampleStateCreateInfo{
					SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
					SampleShadingEnable:  vk.False,
					RasterizationSamples: vk.SampleCount1Bit,
				},
				...
			},
		}
		...
	}()
	...
}
```

Details are available in the [Multisampling part of the Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Multisampling).

### Color blend

The color blend state configures how the pipeline combines the fragment shader results with the data already in the framebuffer.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the info object.
		pipelineInfos := []vk.GraphicsPipelineCreateInfo{
			vk.GraphicsPipelineCreateInfo{
				...
				PColorBlendState: &vk.PipelineColorBlendStateCreateInfo{
					SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
					LogicOpEnable:   vk.False,
					LogicOp:         vk.LogicOpCopy,
					AttachmentCount: 1,
					PAttachments: []vk.PipelineColorBlendAttachmentState{
						vk.PipelineColorBlendAttachmentState{
							ColorWriteMask: vk.ColorComponentFlags(vk.ColorComponentRBit | vk.ColorComponentGBit | vk.ColorComponentBBit | vk.ColorComponentABit),
							BlendEnable:    vk.False,
						},
					},
				},
				...
			},
		}
		...
	}()
	...
}
```

Details are available in the [Color blending part of the Fixed functions](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions#page_Color-blending).

## Creating the pipeline

After building up the create info, I can finish the create pattern calls.

```golang
func NewPipeline(app *TriangleApplication, oldPipeline *Pipeline) *Pipeline {
	...
	pipelines := func() []vk.Pipeline {
		...
		// Create the result object.
		pipelines := make([]vk.Pipeline, len(pipelineInfos))

		// Call the Vulkan function.
		MustSucceed(vk.CreateGraphicsPipelines(app.device,
			vk.PipelineCache(vk.NullHandle),
			1,
			pipelineInfos,
			nil,
			pipelines))

		// Return the pipelines.
		return pipelines
	}()
  ...
}
```

The additional fields in this create call are explained in [Drawing a triangle / Graphics pipeline basics / Conclusion](https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Conclusion).

## Cleanup

The pipelines need to be deleted as part of the Pipeline cleanup.

```golang
func (pipeline *Pipeline) Cleanup(device vk.Device) {
	for _, buffer := range pipeline.SwapchainFramebuffers {
		vk.DestroyFramebuffer(device, buffer, nil)
	}

	for _, pl := range pipeline.Pipelines {
		vk.DestroyPipeline(device, pl, nil)
	}
	...
}
```

## Conclusion

In this part I completed our fixed function configuration, created our shader modules, and created the pipeline handle. I have a pipeline ready for executing commands, and have finished through to [Drawing a triangle / Drawing / Framebuffers](https://vulkan-tutorial.com/Drawing_a_triangle/Drawing/Framebuffers).

In the next part, I will create the command buffers and record commands.

[pipeline.go](https://github.com/ibd1279/vulkangotutorial/blob/5632dcbc7dd4ccc9eebd466aafef13430c8612cf/pipeline.go) ([diff](https://github.com/ibd1279/vulkangotutorial/commit/5632dcbc7dd4ccc9eebd466aafef13430c8612cf#diff-39628d7d0647cdeebd6490044d9c16b508976806719f989b21c058ec71258fe0)) / [shader.vert](https://github.com/ibd1279/vulkangotutorial/blob/5632dcbc7dd4ccc9eebd466aafef13430c8612cf/shaders/shader.vert) / [shader.frag](https://github.com/ibd1279/vulkangotutorial/blob/5632dcbc7dd4ccc9eebd466aafef13430c8612cf/shaders/shader.frag)
