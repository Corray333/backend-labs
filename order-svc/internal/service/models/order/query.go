package order

// QueryOrdersModel represents filter parameters for querying orders
type QueryOrdersModel struct {
	Ids         []int64 `json:"ids,omitempty"`
	CustomerIds []int64 `json:"customerIds,omitempty"`
	Limit       int     `json:"limit,omitempty"`
	Offset      int     `json:"offset,omitempty"`
}
