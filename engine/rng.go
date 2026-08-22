package engine

// The source of variation in the map.
//
// Each generator decision is a hash of the seed, the coordinates it applies to
// and a salt identifying the decision, so a cell can be generated on its own,
// in any order, and give the same result. Chunks therefore need no stitching.

// Salts. Each identifies one decision, so two decisions about the same block do
// not draw the same number.
const (
	saltLayout = iota + 1
	saltOrient
	saltHeight
	saltHue
	saltSaturation
	saltWindowStyle
	saltLit
	saltRoof
	saltNotch
	saltFacade
	saltPond
	saltPlanting
	saltUse
	saltName
	saltMonument
	saltPark
)

// worldSeed shifts every hash, so one set of coordinates gives a different
// city per seed. It is package state: a process generates one world.
var worldSeed int

// SetSeed selects the world to generate. Call it before generating anything.
func SetSeed(n int) { worldSeed = n }

// Seed reports the world in use.
func Seed() int { return worldSeed }

// Hash01 returns a stable value in [0,1) for a set of integer keys.
func Hash01(keys ...int) float64 {
	return float64(hash(keys...)) / 4294967296
}

// HashInt returns a stable value in [0,n) for a set of integer keys.
func HashInt(n int, keys ...int) int {
	if n <= 0 {
		return 0
	}
	return int(hash(keys...) % uint32(n))
}

func hash(keys ...int) uint32 {
	d := uint32(0x9e3779b1) + uint32(int32(worldSeed))*0x27d4eb2f
	for i, k := range keys {
		d += uint32(int32(k))*0x27d4eb2f + uint32(i)*0x165667b1
		d = (d ^ (d >> 13)) * 0x4bf19f61
	}
	d = (d ^ (d >> 15)) * 0x2545f491
	return d ^ (d >> 16)
}
