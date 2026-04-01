package present

type Kind string

const (
	KindInfo    Kind = "info"
	KindSuccess Kind = "success"
	KindWarning Kind = "warning"
	KindError   Kind = "error"
)

type Message struct {
	Kind      Kind
	Title     string
	Body      string
	Ephemeral bool
}

func Info(title, body string, ephemeral bool) Message {
	return Message{Kind: KindInfo, Title: title, Body: body, Ephemeral: ephemeral}
}

func Success(title, body string, ephemeral bool) Message {
	return Message{Kind: KindSuccess, Title: title, Body: body, Ephemeral: ephemeral}
}

func Warning(title, body string, ephemeral bool) Message {
	return Message{Kind: KindWarning, Title: title, Body: body, Ephemeral: ephemeral}
}

func Error(title, body string, ephemeral bool) Message {
	return Message{Kind: KindError, Title: title, Body: body, Ephemeral: ephemeral}
}
