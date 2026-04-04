package interactions

type Kind string

const (
	KindInfo    Kind = "info"
	KindSuccess Kind = "success"
	KindWarning Kind = "warning"
	KindError   Kind = "error"
)
