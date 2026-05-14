package main

import (
	"container/heap"
	"fmt"
	"math"
	"time"
)

type Item struct {
	index    int     // index of the cell in the grid (y * width + x)
	priority float32 // f = g + h
	gScore   float32 // used to break f ties toward straighter paths
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].priority != pq[j].priority {
		return pq[i].priority < pq[j].priority
	}
	// Same f: prefer larger g (smaller h → closer to goal) for cleaner grid paths.
	return pq[i].gScore > pq[j].gScore
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Item))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[0 : n-1]
	return item
}

type AStar struct {
	gridTypes []byte
	gScores   []float32
	parents   []byte
	openSet   PriorityQueue
	closedSet []bool
	width     int
	height    int
	heuristic func(x int, y int, endX int, endY int) float32
	timeTaken time.Duration
}

func (a *AStar) Init(width int, height int) {
	a.gridTypes = make([]byte, width*height)
	a.gScores = make([]float32, width*height)
	a.parents = make([]byte, width*height)
	a.openSet = make(PriorityQueue, 0)
	a.closedSet = make([]bool, width*height)
	a.heuristic = func(x int, y int, endX int, endY int) float32 {
		return float32(math.Abs(float64(x-endX)) + math.Abs(float64(y-endY))) // Manhattan distance default
	}
	a.width = width
	a.height = height

	for i := range a.gScores {
		a.gScores[i] = math.MaxFloat32
	}
}

func (a *AStar) ResetGrid(withTypes bool) {
	for i := range a.gScores {
		if withTypes {
			a.gridTypes[i] = 0
		}
		a.gScores[i] = math.MaxFloat32
		a.parents[i] = 0
		a.closedSet[i] = false
	}
	a.openSet = a.openSet[:0]
}

func (a *AStar) RebuildGrid(width int, height int) {
	a.gridTypes = make([]byte, width*height)
	a.gScores = make([]float32, width*height)
	a.parents = make([]byte, width*height)
	a.openSet = make(PriorityQueue, 0)
	a.closedSet = make([]bool, width*height)
	a.width = width
	a.height = height
}

func (a *AStar) SetHeuristic(heuristic int32) {
	switch heuristic {
	case 0:
		a.heuristic = func(x int, y int, endX int, endY int) float32 {
			return float32(math.Abs(float64(x-endX)) + math.Abs(float64(y-endY))) // Manhattan distance
		}
	case 1:
		a.heuristic = func(x int, y int, endX int, endY int) float32 {
			return float32(math.Sqrt(float64(x-endX)*float64(x-endX) + float64(y-endY)*float64(y-endY))) // Euclidean distance
		}
	case 2:
		a.heuristic = func(x int, y int, endX int, endY int) float32 {
			return float32(math.Max(float64(x-endX), float64(y-endY))) // Chebyshev distance
		}
	}
}

func (a *AStar) SetGridType(x int, y int, gridType byte) {
	/*
		0 = empty
		1 = wall
		2 = start
		3 = end
	*/
	a.gridTypes[y*a.width+x] = gridType
}

func (a *AStar) GetGridType(x int, y int) byte {
	return a.gridTypes[y*a.width+x]
}

func (a *AStar) GetGridTypes() []byte {
	return a.gridTypes
}

func (a *AStar) GetClosedSet() []bool {
	return a.closedSet
}

func (a *AStar) SetGScores(x int, y int, gScore float32) {
	a.gScores[y*a.width+x] = gScore
}

func (a *AStar) GetGScores(x int, y int) float32 {
	return a.gScores[y*a.width+x]
}

func (a *AStar) SetParent(x int, y int, parentx int, parenty int) {
	if parentx < x {
		a.parents[y*a.width+x] = byte(0) // left of the node
	} else if parentx > x {
		a.parents[y*a.width+x] = byte(2) // right of the node
	} else if parenty < y {
		a.parents[y*a.width+x] = byte(1) // above the node
	} else if parenty > y {
		a.parents[y*a.width+x] = byte(3) // below the node
	}
}

func (a *AStar) GetParent(x int, y int) byte {
	return a.parents[y*a.width+x]
}

func (a *AStar) ParentIndexToXY(childx int, childy int, parent byte) (int, int) {
	if parent == 0 {
		return childx - 1, childy // parent left
	} else if parent == 1 {
		return childx, childy - 1 // parent above
	} else if parent == 2 {
		return childx + 1, childy // parent right
	} else if parent == 3 {
		return childx, childy + 1 // parent below
	}
	return childx, childy
}

func (a *AStar) ParentIndexToXYIndex(childx int, childy int, parent byte) int {
	x, y := a.ParentIndexToXY(childx, childy, parent)
	return y*a.width + x
}

func (a *AStar) GetNeighbors(x int, y int) []int {
	neighbors := make([]int, 0)
	if x > 0 {
		neighbors = append(neighbors, y*a.width+x-1)
	}
	if x < a.width-1 {
		neighbors = append(neighbors, y*a.width+x+1)
	}
	if y > 0 {
		neighbors = append(neighbors, y*a.width+x-a.width)
	}
	if y < a.height-1 {
		neighbors = append(neighbors, y*a.width+x+a.width)
	}
	return neighbors
}

func (a *AStar) GetTerrainCost(x int, y int) float32 {
	return 1.0
}

func (a *AStar) GetEvaluatedCells() int {
	cellsEvaluated := 0
	for _, closed := range a.closedSet {
		if closed {
			cellsEvaluated++
		}
	}
	return cellsEvaluated
}

func (a *AStar) GetTimeTaken() time.Duration {
	return a.timeTaken
}

func (a *AStar) CalculatePath(startX int, startY int, endX int, endY int) [][]int {
	timer := time.Now()
	defer func() {
		a.timeTaken = time.Since(timer)
	}()
	startIndex := startY*a.width + startX
	endIndex := endY*a.width + endX
	a.gScores[startIndex] = 0
	startF := a.heuristic(startX, startY, endX, endY)
	heap.Push(&a.openSet, &Item{index: startIndex, priority: startF, gScore: 0})

	for a.openSet.Len() > 0 {
		current := heap.Pop(&a.openSet).(*Item)
		if a.closedSet[current.index] {
			continue
		}
		if current.index == endIndex {
			// We've found the goal!
			fmt.Println("Found the goal!")
			path := make([][]int, 0)
			for currentIndex := current.index; currentIndex != startIndex; currentIndex = a.ParentIndexToXYIndex(currentIndex%a.width, currentIndex/a.width, a.parents[currentIndex]) {
				x, y := a.ParentIndexToXY(currentIndex%a.width, currentIndex/a.width, a.parents[currentIndex])
				path = append(path, []int{x, y})
			}
			return path
		}

		a.closedSet[current.index] = true

		for _, neighborIndex := range a.GetNeighbors(current.index%a.width, current.index/a.width) {
			if a.closedSet[neighborIndex] {
				continue
			}
			if a.gridTypes[neighborIndex] == 1 {
				a.gScores[neighborIndex] = math.MaxFloat32
				continue
			}
			terrainCost := a.GetTerrainCost(neighborIndex%a.width, neighborIndex/a.width)
			tentativeGScore := a.gScores[current.index] + terrainCost
			if tentativeGScore < a.gScores[neighborIndex] {
				a.SetParent(neighborIndex%a.width, neighborIndex/a.width, current.index%a.width, current.index/a.width)
				a.gScores[neighborIndex] = tentativeGScore
				priority := tentativeGScore + a.heuristic(neighborIndex%a.width, neighborIndex/a.width, endX, endY)
				heap.Push(&a.openSet, &Item{index: neighborIndex, priority: priority, gScore: tentativeGScore})
			}
		}
	}
	return make([][]int, 0)
}
