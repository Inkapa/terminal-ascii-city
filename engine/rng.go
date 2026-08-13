package engine

// The source of variation in the map.
//
// Nothing here keeps state. Each generator decision is a hash of the
// coordinates it applies to plus a salt identifying the decision, so a cell can
// be generated on its own, in any order, and give the same result. That is what
// removes the need to stream or stitch chunks.

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
)

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
	d := uint32(0x9e3779b1)
	for i, k := range keys {
		d += uint32(int32(k))*0x27d4eb2f + uint32(i)*0x165667b1
		d = (d ^ (d >> 13)) * 0x4bf19f61
	}
	d = (d ^ (d >> 15)) * 0x2545f491
	return d ^ (d >> 16)
}
