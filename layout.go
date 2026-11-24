package main

// Fit the scene with one scale for both axes. Unusually short/wide panels
// get margins rather than distorted artwork or an unplayable scene.
func sceneViewport(width, height, sceneHeight int) bounds {
	w, h := width, height
	if width*sceneHeight > height*sceneWidth {
		w = height * sceneWidth / sceneHeight
	} else {
		h = width * sceneHeight / sceneWidth
	}
	return bounds{(width - w) / 2, (height - h) / 2, w, h}
}

func (b bounds) scenePoint(p point, height int) point {
	if !b.contains(p) {
		return point{-1, -1}
	}
	return point{(p.x - b.x) * sceneWidth / b.w, (p.y - b.y) * height / b.h}
}
