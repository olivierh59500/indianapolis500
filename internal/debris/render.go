package debris

import "github.com/olivierh59500/indianapolis500/internal/scene"

// Paint is called after the scene and cars, as in the original late effect pass.
func (m *Model) Paint(renderer *scene.Renderer) {
	for _, e := range m.Groups {
		if !e.Visible() {
			continue
		}
		for _, piece := range e.Pieces {
			paintPiece(renderer, e.Points, piece)
		}
	}
	m.Presented()
}

func paintPiece(renderer *scene.Renderer, points [24]scene.Point, piece Piece) {
	switch piece.Mode {
	case Spark:
		renderer.PaintWorldSpark(points[piece.Vertices[0]], piece.Age, piece.Style, piece.Color)
	case Filled, Wire:
		triangle := [3]scene.Point{points[piece.Vertices[0]], points[piece.Vertices[1]], points[piece.Vertices[2]]}
		renderer.PaintWorldTriangle(triangle, piece.Color, piece.Mode == Wire)
	}
}
