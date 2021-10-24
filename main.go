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
