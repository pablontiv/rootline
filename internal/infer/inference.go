package infer

// Inference represents a single inference produced by a category detector.
type Inference struct {
	Type    string `json:"type"`
	Source  string `json:"source"`
	Field   string `json:"field,omitempty"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}
