package orderitem

// QueryOrderItemsModel represents filter parameters for querying order items.
type QueryOrderItemsModel struct {
	Ids               []int64 `json:"ids,omitempty"`
	OrderIds          []int64 `json:"orderIds,omitempty"`
	ProductIds        []int64 `json:"productIds,omitempty"`
	CustomerIds       []int64 `json:"customerIds,omitempty"`
	Page              int     `json:"page,omitempty"`
	PageSize          int     `json:"pageSize,omitempty"`
	IncludeOrderItems bool    `json:"includeOrderItems,omitempty"`
	Limit             int     `json:"limit,omitempty"`
	Offset            int     `json:"offset,omitempty"`
}
