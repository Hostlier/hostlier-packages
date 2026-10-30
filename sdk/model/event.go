package model

// Event reports runtime progress for controller-side sanitization.
type Event struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Step    string `json:"step,omitempty"`
}
