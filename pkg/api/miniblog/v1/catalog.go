package v1

type ReorderCatalogRequest struct {
	Kind       string   `json:"kind"`
	ParentCode string   `json:"parent_code,omitempty"`
	Codes      []string `json:"codes"`
}
