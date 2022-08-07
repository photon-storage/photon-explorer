package mysql

// Config represent root of mysql config
type Config struct {
	Master   connection   `json:"master"`
	Slaves   []connection `json:"slaves"`
	ConnCfg  connCfg      `json:"conn_cfg"`
	LogLevel int          `json:"log_level"`
}

type connection struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
}

type connCfg struct {
	MaxOpenConns int `json:"max_open_conns"`
	MaxIdleConns int `json:"max_idle_conns"`
}
