package main

import (
	"strconv"

	rg "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func canvasMouse() rl.Vector2 {
	return rl.GetMousePosition()
}

func drawGrid(grid [][]int, lineThickness float32) {
	if lineThickness > 5 {
		lineThickness = 0
	}
	for i := 0; i < len(grid); i++ {
		for j := 0; j < len(grid[i]); j++ {
			rl.DrawRectangle(int32(i*25), int32(j*25), 25, 25, rl.NewColor(240, 240, 240, 255))
			if lineThickness > 0 {
				rl.DrawRectangleLinesEx(rl.NewRectangle(float32(i*25), float32(j*25), 25, 25), lineThickness, rl.Black)
			}
		}
	}
}

func generateGrid(width int, height int) [][]int {
	grid := make([][]int, width)
	for i := range grid {
		grid[i] = make([]int, height)
	}
	return grid
}

func main() {
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(int32(800), int32(450), "Raylib - Wayland Safe Zoom & Resize")
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

	grid := generateGrid(width, height)

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
			if camera.Zoom < 0.1 {
				camera.Zoom = 0.1
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
		drawGrid(grid, 1/camera.Zoom)
		rl.EndMode2D()
		rl.EndScissorMode()

		// 2. Draw Sidebar UI
		sidebarX := canvasWidth
		rl.DrawRectangleRec(rl.NewRectangle(sidebarX, 0, sidebarWidth, screenHeight), rl.RayWhite)

		if rg.TextBox(rl.NewRectangle(sidebarX+(10*scale), (10*scale), (80*scale), (20*scale)), &widthInputValue, 20, editModeWidth) {
			editModeWidth = !editModeWidth
			if editModeWidth {
				editModeHeight = false
			}
		}
		rg.Label(rl.NewRectangle(sidebarX+(95*scale), (10*scale), (10*scale), (20*scale)), "x")

		if rg.TextBox(rl.NewRectangle(sidebarX+(110*scale), (10*scale), (80*scale), (20*scale)), &heightInputValue, 20, editModeHeight) {
			editModeHeight = !editModeHeight
			if editModeHeight {
				editModeWidth = false
			}
		}

		if rg.Button(rl.NewRectangle(sidebarX+(10*scale), (40*scale), (180*scale), (30*scale)), "Generate Grid") {
			width, _ = strconv.Atoi(widthInputValue)
			height, _ = strconv.Atoi(heightInputValue)
			grid = generateGrid(width, height)
		}

		rl.EndDrawing()
	}
}
