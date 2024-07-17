package stats

type DBRecord struct {

	// ID is a idempotent identifier derived from other attributes
	// (typically date and query).
	// Please note that the value should not be derived just from
	// the query because we want some variance with repeated queries.
	// (with a single query try, the result can be affected by (bad)luck).
	ID string

	// Datetime specifies date and time when the record was imported
	// (i.e. not benchmarked)
	Datetime int64

	// Query contains the original version of imported query
	Query string

	// QueryNormalized contains the original version of imported query
	QueryNormalized string

	// Corpname is the original corpus query was run with
	Corpname string

	// ProcTime is a procTime reported by KonText in its log.
	// Please note that this time cannot be reliably used to measure
	// queries complexity as most KonText queries run in an async.
	// mode and the query_submit response may occur even if there are
	// no data yet.
	// We store this for further analysis - e.g. if there is at least
	// some correlation between those times and benchmark times.
	ProcTime float64

	// BenchTime is a time of a query measured in a controlled environment.
	// This means running it on MQuery on a computer not doing much other
	// work.
	BenchTime float64

	// TrainingExclude excluded the record from training. Typically, this
	// is for additional validation of the model.
	TrainingExclude bool
}
