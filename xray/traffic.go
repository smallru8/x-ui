package xray

type Traffic struct {
	IsUser bool
	IsInbound bool
	Tag       string
	Up        int64
	Down      int64
}
