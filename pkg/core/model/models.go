package model

// DictItem represents a record in the dictionary table
type DictItem struct {
	DictCode  string `db:"dict_code"`
	ItemValue string `db:"item_value"`
	ItemText  string `db:"item_text"`
}

// MappingStatus defines the state of a field mapping
type MappingStatus int

const (
	StatusUnmapped MappingStatus = iota
	StatusAutoMatched
	StatusManualMapped
)

// FieldMapping relates an ES field to its dictionary status
type FieldMapping struct {
	FieldName  string
	SampleText string
	Status     MappingStatus
	DictCode   string
}

// AggregationResult models a real bucket from ES
type AggBucket struct {
	Key   string `json:"key"`
	Count int    `json:"doc_count"`
}

type AggResult struct {
	Buckets []AggBucket `json:"buckets"`
}
