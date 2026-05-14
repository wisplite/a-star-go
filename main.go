package main

import (
	"image/color"
	"strconv"
	"strings"
	"unsafe"

	rg "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func canvasMouse() rl.Vector2 {
	return rl.GetMousePosition()
}

// Keeps GPU texture uploads in sync with mapImage edits. Full uploads are used after
// path rebuilds; edits use bounding boxes via UpdateTextureRec.
type texSync struct {
	needUpload bool
	uploadFull bool
	partialOk  bool
	x0, y0     int
	x1, y1     int
	rectBuf    []color.RGBA
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (t *texSync) markFull() {
	t.needUpload = true
	t.uploadFull = true
}

func (t *texSync) markRegion(minX, minY, maxX, maxY, gridW, gridH int) {
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= gridW {
		maxX = gridW - 1
	}
	if maxY >= gridH {
		maxY = gridH - 1
	}
	if minX > maxX || minY > maxY {
		return
	}
	t.needUpload = true
	if t.uploadFull {
		return
	}
	if !t.partialOk {
		t.partialOk = true
		t.x0, t.y0 = minX, minY
		t.x1, t.y1 = maxX, maxY
		return
	}
	if minX < t.x0 {
		t.x0 = minX
	}
	if minY < t.y0 {
		t.y0 = minY
	}
	if maxX > t.x1 {
		t.x1 = maxX
	}
	if maxY > t.y1 {
		t.y1 = maxY
	}
}

func (t *texSync) flush(img *rl.Image, tex *rl.Texture2D, gridW, gridH int) {
	if !t.needUpload {
		return
	}
	if t.uploadFull {
		ptr := (*color.RGBA)(unsafe.Pointer(img.Data))
		pixels := unsafe.Slice(ptr, gridW*gridH)
		rl.UpdateTexture(*tex, pixels)
		t.uploadFull = false
		t.partialOk = false
	} else if t.partialOk {
		w := t.x1 - t.x0 + 1
		h := t.y1 - t.y0 + 1
		n := w * h
		if cap(t.rectBuf) < n {
			t.rectBuf = make([]color.RGBA, n)
		} else {
			t.rectBuf = t.rectBuf[:n]
		}
		ptr := (*color.RGBA)(unsafe.Pointer(img.Data))
		full := unsafe.Slice(ptr, gridW*gridH)
		for row := 0; row < h; row++ {
			src := (t.y0+row)*gridW + t.x0
			copy(t.rectBuf[row*w:], full[src:src+w])
		}
		rec := rl.NewRectangle(float32(t.x0), float32(t.y0), float32(w), float32(h))
		rl.UpdateTextureRec(*tex, rec, t.rectBuf)
		t.partialOk = false
	}
	t.needUpload = false
}

// paintWallLine updates both mapImage and the A* grid for every grid cell crossed by the segment.
func paintWallLine(a *AStar, mapImage *rl.Image, x0, y0, x1, y1, gridW, gridH int, col color.RGBA, tex *texSync) {
	minX := x0
	if x1 < minX {
		minX = x1
	}
	maxX := x0
	if x1 > maxX {
		maxX = x1
	}
	minY := y0
	if y1 < minY {
		minY = y1
	}
	maxY := y0
	if y1 > maxY {
		maxY = y1
	}
	tex.markRegion(minX, minY, maxX, maxY, gridW, gridH)

	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := 1
	sy := 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		if x0 >= 0 && x0 < gridW && y0 >= 0 && y0 < gridH {
			switch a.GetGridType(x0, y0) {
			case 2, 3: // start / end
			default:
				rl.ImageDrawPixel(mapImage, int32(x0), int32(y0), col)
				a.SetGridType(x0, y0, 1)
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// paintEraseLine clears wall cells along the segment (same grid traversal as paintWallLine).
func paintEraseLine(a *AStar, mapImage *rl.Image, x0, y0, x1, y1, gridW, gridH int, empty color.RGBA, tex *texSync) {
	minX := x0
	if x1 < minX {
		minX = x1
	}
	maxX := x0
	if x1 > maxX {
		maxX = x1
	}
	minY := y0
	if y1 < minY {
		minY = y1
	}
	maxY := y0
	if y1 > maxY {
		maxY = y1
	}
	tex.markRegion(minX, minY, maxX, maxY, gridW, gridH)

	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := 1
	sy := 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		if x0 >= 0 && x0 < gridW && y0 >= 0 && y0 < gridH {
			switch a.GetGridType(x0, y0) {
			case 2, 3: // start / end
			default:
				rl.ImageDrawPixel(mapImage, int32(x0), int32(y0), empty)
				a.SetGridType(x0, y0, 0)
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
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
	generateGridError := false

	toolOptions := []string{"Wall", "Start", "End", "Erase"}
	toolOptionsText := strings.Join(toolOptions, ";")
	activeTool := int32(0)
	toolDropdownOpen := false
	var tex texSync // GPU uploads: partial rects while painting; full after path resets

	heuristicOptions := []string{"Manhattan", "Euclidean", "Chebyshev"}
	heuristicOptionsText := strings.Join(heuristicOptions, ";")
	activeHeuristic := int32(0)
	heuristicDropdownOpen := false

	cellSize := float32(25)

	lastMousePos := rl.NewVector2(-1, -1)
	startPos := rl.NewVector2(-1, -1)
	endPos := rl.NewVector2(-1, -1)

	mapImage := rl.GenImageColor(width, height, rl.NewColor(240, 240, 240, 255))
	mapTexture := rl.LoadTextureFromImage(mapImage)
	defer rl.UnloadTexture(mapTexture)
	defer rl.UnloadImage(mapImage)

	astar := AStar{}
	astar.Init(width, height)

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

		// Paint logic
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			mousePos := rl.GetMousePosition()
			if int(mousePos.X) < int(canvasWidth) {
				worldPos := rl.GetScreenToWorld2D(rl.GetMousePosition(), camera)
				x := int(worldPos.X / cellSize)
				y := int(worldPos.Y / cellSize)
				if x >= 0 && x < width && y >= 0 && y < height {
					// mapImage is one pixel per cell; drawing uses cell indices, not raw world coords.
					switch activeTool {
					case 0: // Wall — Bresenham line into image + simulator grid (skips start/end cells)
						wallCol := color.RGBA{A: 255}
						if lastMousePos.X != -1 && lastMousePos.Y != -1 {
							prevX := int(lastMousePos.X / cellSize)
							prevY := int(lastMousePos.Y / cellSize)
							paintWallLine(&astar, mapImage, prevX, prevY, x, y, width, height, wallCol, &tex)
						} else {
							paintWallLine(&astar, mapImage, x, y, x, y, width, height, wallCol, &tex)
						}
					case 1: // Start — must run on first paint frame (lastMousePos may be -1 after release)
						if int(startPos.X) != x || int(startPos.Y) != y {
							if startPos.X >= 0 && startPos.Y >= 0 {
								ox, oy := int(startPos.X), int(startPos.Y)
								rl.ImageDrawPixel(mapImage, int32(startPos.X), int32(startPos.Y), rl.NewColor(240, 240, 240, 255))
								tex.markRegion(ox, oy, ox, oy, width, height)
							}
							rl.ImageDrawPixel(mapImage, int32(x), int32(y), rl.NewColor(0, 255, 0, 255))
							tex.markRegion(x, y, x, y, width, height)
							startPos = rl.NewVector2(float32(x), float32(y))
							astar.SetGridType(x, y, 2)
						}
					case 2: // End
						if int(endPos.X) != x || int(endPos.Y) != y {
							if endPos.X >= 0 && endPos.Y >= 0 {
								ox, oy := int(endPos.X), int(endPos.Y)
								rl.ImageDrawPixel(mapImage, int32(endPos.X), int32(endPos.Y), rl.NewColor(240, 240, 240, 255))
								tex.markRegion(ox, oy, ox, oy, width, height)
							}
							rl.ImageDrawPixel(mapImage, int32(x), int32(y), rl.NewColor(255, 0, 0, 255))
							tex.markRegion(x, y, x, y, width, height)
							endPos = rl.NewVector2(float32(x), float32(y))
							astar.SetGridType(x, y, 3)
						}
					case 3: // Erase — same cell indices as walls; ImageDrawLine used world coords before (wrong space).
						emptyCol := color.RGBA{R: 240, G: 240, B: 240, A: 255}
						if lastMousePos.X != -1 && lastMousePos.Y != -1 {
							prevX := int(lastMousePos.X / cellSize)
							prevY := int(lastMousePos.Y / cellSize)
							paintEraseLine(&astar, mapImage, prevX, prevY, x, y, width, height, emptyCol, &tex)
						} else {
							paintEraseLine(&astar, mapImage, x, y, x, y, width, height, emptyCol, &tex)
						}
					}
					lastMousePos = worldPos
				}
			}
		}

		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			lastMousePos = rl.NewVector2(-1, -1)
		}

		// --- DRAWING ---
		rl.BeginDrawing()
		rl.ClearBackground(rl.White)

		// 1. Draw Canvas
		rl.BeginScissorMode(0, 0, int32(canvasWidth), int32(screenHeight))
		rl.BeginMode2D(camera)
		tex.flush(mapImage, &mapTexture, width, height)
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

		if generateGridError {
			result := rg.MessageBox(rl.NewRectangle(screenWidth/2-(100*scale), screenHeight/2-(40*scale), (200*scale), (120*scale)), "Error", "Width and/or height must be\nless than 16000", "OK")
			if result >= 0 {
				generateGridError = false
			}
		}

		// Generate Grid Button
		if rg.Button(rl.NewRectangle(sidebarX+(10*scale), (40*scale), (180*scale), (30*scale)), "Generate Grid") {
			width, _ = strconv.Atoi(widthInputValue)
			height, _ = strconv.Atoi(heightInputValue)
			if width > 16000 || height > 16000 {
				generateGridError = true
			} else {
				rl.UnloadTexture(mapTexture)
				rl.UnloadImage(mapImage)
				mapImage = rl.GenImageColor(width, height, rl.NewColor(240, 240, 240, 255))
				mapTexture = rl.LoadTextureFromImage(mapImage)
				astar.RebuildGrid(width, height)
				startPos = rl.NewVector2(-1, -1)
				endPos = rl.NewVector2(-1, -1)
				tex = texSync{rectBuf: tex.rectBuf}
			}
		}

		// Tool Selector (text must be "opt1;opt2;..." — raygui splits on ';' and needs 2+ items)
		rg.Label(rl.NewRectangle(sidebarX+(10*scale), (75*scale), (180*scale), (30*scale)), "Tool:")
		if rg.DropdownBox(rl.NewRectangle(sidebarX+(10*scale), (100*scale), (180*scale), (30*scale)), toolOptionsText, &activeTool, toolDropdownOpen) {
			toolDropdownOpen = !toolDropdownOpen
		}

		// Heuristic Selector
		if !toolDropdownOpen {
			rg.Label(rl.NewRectangle(sidebarX+(10*scale), (135*scale), (180*scale), (30*scale)), "Heuristic:")
			if rg.DropdownBox(rl.NewRectangle(sidebarX+(10*scale), (160*scale), (180*scale), (30*scale)), heuristicOptionsText, &activeHeuristic, heuristicDropdownOpen) {
				heuristicDropdownOpen = !heuristicDropdownOpen
			}
		}

		// Reset Visualization Button
		if rg.Button(rl.NewRectangle(sidebarX+(10*scale), (screenHeight-(80*scale)), (180*scale), (30*scale)), "Reset Visualization") {
			astar.ResetGrid(false) // keep grid types, otherwise it will delete the board before simulating
			gridTypes := astar.GetGridTypes()
			for i, gridType := range gridTypes {
				// reset the map image
				switch gridType {
				case 0:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(240, 240, 240, 255))
				case 1:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(0, 0, 0, 255))
				case 2:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(0, 255, 0, 255))
				case 3:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(255, 0, 0, 255))
				}
			}
			tex.markFull()
		}

		// Calculate Path Button
		if rg.Button(rl.NewRectangle(sidebarX+(10*scale), (screenHeight-(40*scale)), (180*scale), (30*scale)), "Calculate Path") {
			astar.ResetGrid(false) // keep grid types, otherwise it will delete the board before simulating
			astar.SetHeuristic(activeHeuristic)
			gridTypes := astar.GetGridTypes()
			for i, gridType := range gridTypes {
				// reset the map image
				switch gridType {
				case 0:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(240, 240, 240, 255))
				case 1:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(0, 0, 0, 255))
				case 2:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(0, 255, 0, 255))
				case 3:
					rl.ImageDrawPixel(mapImage, int32(i%width), int32(i/width), rl.NewColor(255, 0, 0, 255))
				}
			}
			path := astar.CalculatePath(int(startPos.X), int(startPos.Y), int(endPos.X), int(endPos.Y))
			closedSet := astar.GetClosedSet()
			for i, closed := range closedSet {
				if closed {
					x := i % width
					y := i / width
					if x != int(startPos.X) || y != int(startPos.Y) {
						rl.ImageDrawPixel(mapImage, int32(x), int32(y), rl.NewColor(0, 0, 255, 255))
					}
				}
			}
			for _, p := range path {
				if p[0] != int(startPos.X) || p[1] != int(startPos.Y) { // we want to keep the start position green
					rl.ImageDrawPixel(mapImage, int32(p[0]), int32(p[1]), rl.NewColor(255, 255, 0, 255))
				}
			}
			tex.markFull()
		}

		rl.EndDrawing()
	}
}
