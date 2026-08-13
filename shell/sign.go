package shell

// The lettering used on mounted signs: a 3x5 bitmap per character, assembled
// into a word grid once at start-up. Signs cycle through the words on a clock.

var signFont = map[byte][5]string{
	'A': {".#.", "#.#", "###", "#.#", "#.#"},
	'B': {"##.", "#.#", "##.", "#.#", "##."},
	'C': {".##", "#..", "#..", "#..", ".##"},
	'D': {"##.", "#.#", "#.#", "#.#", "##."},
	'E': {"###", "#..", "##.", "#..", "###"},
	'F': {"###", "#..", "##.", "#..", "#.."},
	'H': {"#.#", "#.#", "###", "#.#", "#.#"},
	'I': {"###", ".#.", ".#.", ".#.", "###"},
	'K': {"#.#", "#.#", "##.", "#.#", "#.#"},
	'L': {"#..", "#..", "#..", "#..", "###"},
	'N': {"#.#", "###", "###", "###", "#.#"},
	'O': {".#.", "#.#", "#.#", "#.#", ".#."},
	'P': {"##.", "#.#", "##.", "#..", "#.."},
	'R': {"##.", "#.#", "##.", "#.#", "#.#"},
	'S': {".##", "#..", ".#.", "..#", "##."},
	'T': {"###", ".#.", ".#.", ".#.", ".#."},
	'X': {"#.#", "#.#", ".#.", "#.#", "#.#"},
}

// signWords are the words a sign can read. Four letters each, so a word fills
// the board exactly.
var signWords = []string{"CAFE", "FOOD", "OPEN", "SALE", "TAXI", "BANK", "SHOP", "BEER"}

var (
	wordGrids   = buildWordGrids()
	letterGrids = buildLetterGrids()
)

// signWordGrid returns the 5x15 bitmap of a whole word.
func signWordGrid(i int) []string { return wordGrids[i%len(wordGrids)] }

// signLetterGrid returns the 5x15 bitmap of a word's initial, centred.
func signLetterGrid(i int) []string { return letterGrids[i%len(letterGrids)] }

func buildWordGrids() [][]string {
	out := make([][]string, len(signWords))
	for w, word := range signWords {
		rows := make([]string, 5)
		for i := 0; i < len(word); i++ {
			g := glyphFor(word[i])
			for r := 0; r < 5; r++ {
				if i > 0 {
					rows[r] += "."
				}
				rows[r] += g[r]
			}
		}
		out[w] = rows
	}
	return out
}

func buildLetterGrids() [][]string {
	out := make([][]string, len(signWords))
	for w, word := range signWords {
		g := glyphFor(word[0])
		rows := make([]string, 5)
		for r := 0; r < 5; r++ {
			rows[r] = "......" + g[r] + "......"
		}
		out[w] = rows
	}
	return out
}

func glyphFor(c byte) [5]string {
	if g, ok := signFont[c]; ok {
		return g
	}
	return [5]string{"...", "...", "...", "...", "..."}
}
