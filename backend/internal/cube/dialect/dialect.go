package dialect

type Dialect interface {
	Quote(identifier string) string
	EscapeString(s string) string
}

type Postgres struct{}

func (Postgres) Quote(identifier string) string {
	return "\"" + identifier + "\""
}

func (Postgres) EscapeString(s string) string {
	return s
}
