package repositories

import "go.mongodb.org/mongo-driver/bson"

// Scope names the one Branch of one Client that a read or write may touch.
type Scope struct {
	ClientId string
	BranchId string
}

// filter returns a new filter holding fields plus this Scope's Branch.
// The argument is not modified.
func (s Scope) filter(fields bson.M) bson.M {
	scoped := make(bson.M, len(fields)+1)
	for key, value := range fields {
		scoped[key] = value
	}
	scoped["branchId"] = s.BranchId
	return scoped
}
