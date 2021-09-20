package server

type HistoryLog struct {
	Order   int      `json:"order"`
	Message *Message `json:"message"`
}
