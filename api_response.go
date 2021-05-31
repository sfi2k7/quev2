package quev2

type ApiResponse struct {
	Result  []string
	Success bool
	Error   string
	Took    string
}

type ApiItemResponse struct {
	Result  []*Item
	Success bool
	Error   string
	Took    string
}
