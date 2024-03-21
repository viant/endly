package transfer

// Asset represents a workflow asset
type Asset struct {
	//ID represents asset ID
	ID string
	//Location represents asset location
	Location string
	//Description represents asset description
	Description string
	//WorkflowID represents workflow ID
	WorkflowID string
	//TagID represents tag ID
	TagID string
	//Index represents asset index
	Index int
	//Source represents asset source
	Source string
	//Format represents asset format
	Format string
	//Codec represents asset codec (i.e. gzip, zstd)
	Codec string
}
