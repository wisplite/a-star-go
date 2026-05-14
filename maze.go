package main

import (
	"math/rand"
	"time"
)

func (a *AStar) GenerateMaze(mazeType int32) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	switch mazeType {
	case 0:
		a.GenerateRecursiveDivisionMaze(r)
	case 1:
		a.GenerateIterativeDFSMaze(r)
	case 2:
		a.GenerateCellularAutomataMaze(r)
	}
}

func (a *AStar) GenerateRecursiveDivisionMaze(r *rand.Rand) {
	// 1. Start with an entirely empty grid
	for i := range a.gridTypes {
		a.gridTypes[i] = 0
	}

	// 2. Draw outer boundary walls
	for x := 0; x < a.width; x++ {
		a.gridTypes[x] = 1
		a.gridTypes[(a.height-1)*a.width+x] = 1
	}
	for y := 0; y < a.height; y++ {
		a.gridTypes[y*a.width] = 1
		a.gridTypes[y*a.width+(a.width-1)] = 1
	}

	// 3. Declare the recursive closure
	var divide func(x, y, w, h int)
	divide = func(x, y, w, h int) {
		// Base case: room is too small to divide safely
		if w <= 3 || h <= 3 {
			return
		}

		// Choose orientation based on proportions to keep rooms somewhat square
		horizontal := h > w
		if w == h {
			horizontal = r.Intn(2) == 0
		}

		if horizontal {
			// Horizontal Wall: Must be on an EVEN local Y coordinate
			wallY := (r.Intn((h-2)/2) * 2) + 2
			// Gap (door): Must be on an ODD local X coordinate
			gapX := (r.Intn((w-1)/2) * 2) + 1

			for px := 0; px < w; px++ {
				if px != gapX {
					a.gridTypes[(y+wallY)*a.width+(x+px)] = 1
				}
			}
			// Recurse top and bottom
			divide(x, y, w, wallY)
			divide(x, y+wallY, w, h-wallY)

		} else {
			// Vertical Wall: Must be on an EVEN local X coordinate
			wallX := (r.Intn((w-2)/2) * 2) + 2
			// Gap (door): Must be on an ODD local Y coordinate
			gapY := (r.Intn((h-1)/2) * 2) + 1

			for py := 0; py < h; py++ {
				if py != gapY {
					a.gridTypes[(y+py)*a.width+(x+wallX)] = 1
				}
			}
			// Recurse left and right
			divide(x, y, wallX, h)
			divide(x+wallX, y, w-wallX, h)
		}
	}

	// Start the recursion on the inside of the boundary walls
	divide(1, 1, a.width-2, a.height-2)
}

func (a *AStar) GenerateIterativeDFSMaze(r *rand.Rand) {
	// 1. Fill the entire grid with walls (type 1)
	for i := range a.gridTypes {
		a.gridTypes[i] = 1
	}

	// 2. The Stack (We use a Go slice instead of recursion to prevent Stack Overflow)
	stack := make([]int, 0)

	// 3. Pick a random starting cell (MUST be odd coordinates for the step-by-two math)
	startX := (r.Intn(a.width/2) * 2) + 1
	startY := (r.Intn(a.height/2) * 2) + 1

	// Bounds check just in case
	if startX >= a.width {
		startX = a.width - 2
	}
	if startY >= a.height {
		startY = a.height - 2
	}

	startIdx := startY*a.width + startX
	a.gridTypes[startIdx] = 0 // Carve the first floor
	stack = append(stack, startIdx)

	// Directions for stepping by TWO (Up, Down, Left, Right)
	dirs := [][]int{{0, -2}, {0, 2}, {-2, 0}, {2, 0}}

	// 4. The DFS Loop
	for len(stack) > 0 {
		// Pop the top of the stack
		currentIdx := stack[len(stack)-1]
		cx := currentIdx % a.width
		cy := currentIdx / a.width

		// Find all valid, unvisited neighbors (distance 2)
		validNeighbors := make([][]int, 0)
		for _, dir := range dirs {
			nx, ny := cx+dir[0], cy+dir[1]
			// Check bounds
			if nx > 0 && nx < a.width-1 && ny > 0 && ny < a.height-1 {
				// If it's still a wall, we haven't visited it yet
				if a.gridTypes[ny*a.width+nx] == 1 {
					validNeighbors = append(validNeighbors, []int{nx, ny, dir[0], dir[1]})
				}
			}
		}

		if len(validNeighbors) > 0 {
			// Pick a random valid neighbor
			next := validNeighbors[r.Intn(len(validNeighbors))]
			nx, ny, dx, dy := next[0], next[1], next[2], next[3]

			// Carve the neighbor (distance 2)
			a.gridTypes[ny*a.width+nx] = 0

			// Carve the wall BETWEEN current and neighbor (distance 1)
			wallX, wallY := cx+(dx/2), cy+(dy/2)
			a.gridTypes[wallY*a.width+wallX] = 0

			// Push the neighbor to the stack
			stack = append(stack, ny*a.width+nx)
		} else {
			// Backtrack! No valid neighbors, so pop it permanently
			stack = stack[:len(stack)-1]
		}
	}
}

func (a *AStar) GenerateCellularAutomataMaze(r *rand.Rand) {
	// 1. Initial State: Fill with random noise (approx 45% walls)
	for i := range a.gridTypes {
		// Leave the edges as walls to contain the caves
		x := i % a.width
		y := i / a.width
		if x == 0 || x == a.width-1 || y == 0 || y == a.height-1 {
			a.gridTypes[i] = 1
		} else if r.Float32() < 0.45 {
			a.gridTypes[i] = 1
		} else {
			a.gridTypes[i] = 0
		}
	}

	// 2. The Smoothing Passes (5 iterations is usually the sweet spot)
	buffer := make([]byte, a.width*a.height)

	for step := 0; step < 5; step++ {
		for y := 0; y < a.height; y++ {
			for x := 0; x < a.width; x++ {
				wallCount := 0

				// Count the 8 surrounding neighbors
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := x+dx, y+dy

						// Edges of the map count as walls
						if nx < 0 || nx >= a.width || ny < 0 || ny >= a.height {
							wallCount++
						} else if a.gridTypes[ny*a.width+nx] == 1 {
							wallCount++
						}
					}
				}

				// The Automata Rules:
				// If surrounded by walls, become a wall.
				// If surrounded by empty space, become empty.
				idx := y*a.width + x
				if wallCount > 4 {
					buffer[idx] = 1
				} else if wallCount < 4 {
					buffer[idx] = 0
				} else {
					buffer[idx] = a.gridTypes[idx] // Stays the same
				}
			}
		}
		// Copy the buffer back to the main grid for the next pass
		copy(a.gridTypes, buffer)
	}
}
