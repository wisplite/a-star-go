package main

import (
	"strconv"
	"strings"

	rg "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func canvasMouse() rl.Vector2 {
	return rl.GetMousePosition()
}

func drawInfiniteGridLines(camera rl.Camera2D, canvasW float32, canvasH float32, cellSize float32, width int, height int) {
	// 1. Level of Detail (LOD) Check
	// If we are zoomed out too far, don't draw the gridlines at all
	if camera.Zoom < 0.4 {
		return
	}

	lineThickness := 1.0 / camera.Zoom
	lineColor := rl.NewColor(200, 200, 200, 255) // Light gray so it's not distracting

	// 2. Draw Vertical Lines
	// Start at the left edge of the screen, draw a line from top to bottom,
	// step right by cellSize, repeat until off the right edge.
	for x := float32(0.0); x <= cellSize*float32(width); x += cellSize {
		rl.DrawLineEx(
			rl.NewVector2(x, 0),
			rl.NewVector2(x, cellSize*float32(height)),
			lineThickness,
			lineColor,
		)
	}

	// 3. Draw Horizontal Lines
	// Start at the top edge of the screen, draw a line from left to right,
	// step down by cellSize, repeat until off the bottom edge.
	for y := float32(0.0); y <= cellSize*float32(height); y += cellSize {
		rl.DrawLineEx(
			rl.NewVector2(0, y),
			rl.NewVector2(cellSize*float32(width), y),
			lineThickness,
			lineColor,
		)
	}
}

func main() {
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(int32(800), int32(450), "A* Visualizer")
	defer rl.CloseWindow()

	scale := rl.GetWindowScaleDPI().X
	if scale == 0 {
		scale = 2.0 // Fallback value
	}

	rg.SetStyle(rg.DEFAULT, rg.TEXT_SIZE, rg.PropertyValue(int64(10*scale)))
	rl.SetTargetFPS(60)

	// Keep offset safely at 0,0. We will never modify this during runtime.
	camera := rl.Camera2D{
		Offset:   rl.NewVector2(0, 0),
		Target:   rl.NewVector2(0, 0),
		Rotation: 0.0,
		Zoom:     1.0,
	}

	width := 10
	height := 10
	widthInputValue := "10"
	heightInputValue := "10"
	editModeWidth := false
	editModeHeight := false

	toolOptions := []string{"Wall", "Start", "End"}
	toolOptionsText := strings.Join(toolOptions, ";")
	activeTool := int32(0)
	toolDropdownOpen := false

	cellSize := float32(25)

	mapImage := rl.GenImageColor(width, height, rl.NewColor(240, 240, 240, 255))
	mapTexture := rl.LoadTextureFromImage(mapImage)
	defer rl.UnloadTexture(mapTexture)
	defer rl.UnloadImage(mapImage)

	for !rl.WindowShouldClose() {
		screenWidth := float32(rl.GetScreenWidth())
		screenHeight := float32(rl.GetScreenHeight())

		sidebarWidth := float32(200) * scale
		if sidebarWidth > screenWidth {
			sidebarWidth = screenWidth
		}
		canvasWidth := screenWidth - sidebarWidth

		wheel := rl.GetMouseWheelMove()

		// --- THE BULLETPROOF ZOOM MATH ---
		if wheel != 0 {
			mousePos := canvasMouse()

			// 1. Where is the mouse in the world BEFORE zooming?
			worldPosBefore := rl.GetScreenToWorld2D(mousePos, camera)

			// 2. Apply the zoom
			camera.Zoom += float32(wheel) * 0.1
			if camera.Zoom < 0.01 {
				camera.Zoom = 0.01
			}

			// 3. Where does that same screen pixel point to AFTER zooming?
			worldPosAfter := rl.GetScreenToWorld2D(mousePos, camera)

			// 4. Shift the camera target to compensate for the difference
			camera.Target.X += (worldPosBefore.X - worldPosAfter.X)
			camera.Target.Y += (worldPosBefore.Y - worldPosAfter.Y)
		}

		// Move the camera with the right mouse button
		if rl.IsMouseButtonDown(rl.MouseRightButton) {
			delta := rl.GetMouseDelta()
			camera.Target.X -= delta.X / camera.Zoom
			camera.Target.Y -= delta.Y / camera.Zoom
		}

		// --- DRAWING ---
		rl.BeginDrawing()
		rl.ClearBackground(rl.White)

		// 1. Draw Canvas
		rl.BeginScissorMode(0, 0, int32(canvasWidth), int32(screenHeight))
		rl.BeginMode2D(camera)
		rl.DrawTextureEx(
			mapTexture,
			rl.NewVector2(0, 0), // Position
			0.0,                 // Rotation
			cellSize,            // Scale factor
			rl.White,            // Tint (White means no tint)
		)
		drawInfiniteGridLines(camera, canvasWidth, screenHeight, cellSize, width, height)
		rl.EndMode2D()
		rl.EndScissorMode()

		// 2. Draw Sidebar UI
		sidebarX := canvasWidth
		rl.DrawRectangleRec(rl.NewRectangle(sidebarX, 0, sidebarWidth, screenHeight), rl.RayWhite)

		// Width Input
		if rg.TextBox(rl.NewRectangle(sidebarX+(10*scale), (10*scale), (80*scale), (20*scale)), &widthInputValue, 20, editModeWidth) {
			editModeWidth = !editModeWidth
			if editModeWidth {
				editModeHeight = false
			}
		}
		rg.Label(rl.NewRectangle(sidebarX+(95*scale), (10*scale), (10*scale), (20*scale)), "x")

		// Height Input
		if rg.TextBox(rl.NewRectangle(sidebarX+(110*scale), (10*scale), (80*scale), (20*scale)), &heightInputValue, 20, editModeHeight) {
			editModeHeight = !editModeHeight
			if editModeHeight {
				editModeWidth = false
			}
		}

		// Generate Grid Button
		if rg.Button(rl.NewRectangle(sidebarX+(10*scale), (40*scale), (180*scale), (30*scale)), "Generate Grid") {
			width, _ = strconv.Atoi(widthInputValue)
			height, _ = strconv.Atoi(heightInputValue)
			rl.UnloadTexture(mapTexture)
			rl.UnloadImage(mapImage)
			mapImage = rl.GenImageColor(width, height, rl.NewColor(240, 240, 240, 255))
			mapTexture = rl.LoadTextureFromImage(mapImage)
			defer rl.UnloadTexture(mapTexture)
			defer rl.UnloadImage(mapImage)
		}

		// Tool Selector (text must be "opt1;opt2;..." — raygui splits on ';' and needs 2+ items)
		rg.Label(rl.NewRectangle(sidebarX+(10*scale), (75*scale), (180*scale), (30*scale)), "Tool:")
		if rg.DropdownBox(rl.NewRectangle(sidebarX+(10*scale), (100*scale), (180*scale), (30*scale)), toolOptionsText, &activeTool, toolDropdownOpen) {
			toolDropdownOpen = !toolDropdownOpen
		}
		rl.EndDrawing()
	}
}
