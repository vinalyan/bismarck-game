package services

// DetectionTarget описывает объект, для которого фиксируется смена статуса обнаружения.
type DetectionTarget struct {
	ID       string
	Name     string
	Owner    string
	Position string
	Type     string // например, "unit" или "task_force"
}
