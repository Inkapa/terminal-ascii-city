package engine

// What a building is, and what its sign says.
//
// This is map data, not a rendering choice: the frontage outside and the
// fittings inside both need it, and they have to agree. Hashed from the block
// and the plot index, so it is never stored.

// A use is what a building is for, and the word its sign ends with.
type use struct {
	ID   string
	Sign string
	Hues [3]uint16 // the colours the inside is fitted out in
}

var uses = [8]use{
	{"retail", "SUPPLY", [3]uint16{165, 190, 42}},
	{"cafe", "CAFE", [3]uint16{28, 42, 175}},
	{"office", "OFFICES", [3]uint16{205, 188, 220}},
	{"clinic", "CLINIC", [3]uint16{175, 195, 45}},
	{"workshop", "WORKS", [3]uint16{28, 12, 195}},
	{"lobby", "HOUSE", [3]uint16{42, 188, 292}},
	{"laundrette", "LAUNDRY", [3]uint16{190, 205, 48}},
	{"arcade", "ARCADE", [3]uint16{292, 205, 48}},
}

// signPrefixes are the ordinary place words a business is named after.
var signPrefixes = [8]string{
	"CENTRAL", "NORTH", "EAST", "WEST", "RIVER", "STATION", "MARKET", "HARBOUR",
}

// Identity returns what stands on one plot: its trade, the words on its sign
// and the colours it is fitted out in.
func Identity(bx, bz, slot int) Descriptor {
	u := uses[HashInt(len(uses), bx, bz, slot, saltUse)]
	return Descriptor{
		Label:     signPrefixes[HashInt(len(signPrefixes), bx, bz, slot, saltName)] + " " + u.Sign,
		Archetype: u.ID,
		TypeLabel: u.Sign,
		Palette:   u.Hues,
	}
}
