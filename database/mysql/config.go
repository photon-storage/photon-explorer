package mysql

// Config defines mysql configuration.
type Config struct {
	Master       connection   `json:"master"`
	Slaves       []connection `json:"slaves"`
	MaxOpenConns int          `json:"max_open_conns"`
	MaxIdleConns int          `json:"max_idle_conns"`
	LogLevel     string       `json:"log_level"`
}

type connection struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
}
