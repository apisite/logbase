package logbase

type LogType int

const (
    Nginx LogType = iota + 1
    Pg
    Journal
)

func (lt LogType) String() string {
    return [...]string{"nginx", "pg", "journal"}[lt-1]
}
