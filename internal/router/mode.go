package router

// Mode represents a routing mode
type Mode string

const (
	ModeDirect Mode = "direct"
	ModeWarp   Mode = "warp"
	ModeHome   Mode = "home"
)

// String returns the mode as a string
func (m Mode) String() string {
	return string(m)
}

// IsValid checks if the mode is valid
func (m Mode) IsValid() bool {
	switch m {
	case ModeDirect, ModeWarp, ModeHome:
		return true
	default:
		return false
	}
}
