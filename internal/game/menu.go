package game

// menuItemAt uses the same item rows as the visible menu. Mouse input stays
// confined to the application and needs no pointer capture.
func (g *Game) menuItemAt(x, y int) (int, bool) {
	if x < 40 || x >= 280 {
		return 0, false
	}
	var rows []int
	switch g.menuPage {
	case mainMenuPage:
		rows = []int{56, 64, 88, 96, 104, 128, 136, 144}
	case raceMenuPage:
		rows = []int{72, 104, 128, 144}
	case teamMenuPage:
		rows = []int{80, 96, 112}
	case fileMenuPage:
		rows = []int{64, 72, 88, 96, 112, 120, 136, 144}
	case optionsMenuPage:
		rows = []int{72, 112}
	}
	for index, row := range rows {
		if y >= row && y < row+8 {
			return index, true
		}
	}
	return 0, false
}
