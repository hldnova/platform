package execute

type StreamContext interface {
	Bounds() Bounds
	UpdateBounds(Bounds)
}
