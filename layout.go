package main

func titleLogoBounds(height int) bounds {
	floor := (height/2)*2 - floorHeight
	return bounds{(sceneWidth - logoWidth) / 2, floor * 21 / 100, logoWidth, logoHeight}
}

func titleLetteringBounds(height int) bounds {
	logo := titleLogoBounds(height)
	// logo.png's cream lettering occupies (2,3)-(88,18) inside its 89x24
	// canvas. The menu draws it at 2x; match the ink, not transparent padding.
	return bounds{logo.x + 4, logo.y + 6, 172, 30}
}

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
