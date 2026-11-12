package repositories

// Scope names the one Branch of one Client that a read or write may touch.
type Scope struct {
	ClientId string
	BranchId string
}
