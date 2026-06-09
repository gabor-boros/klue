package diagnose

type Suggestion struct {
	Title       string `json:"title"`
	Command     string `json:"command,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}
