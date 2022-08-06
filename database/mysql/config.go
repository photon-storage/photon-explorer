package mysql

// Config represent root of mysql config
type Config struct {
	Master   Connection   `json:"master"`
	Slaves   []Connection `json:"slaves"`
	ConnCfg  ConnCfg      `json:"conn_cfg"`
	LogLevel int          `json:"log_level"`
}

// Connection represent the mysql connection config
type Connection struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
}

// ConnCfg represent mysql connection config
type ConnCfg struct {
	MaxOpenConns int `json:"max_open_conns"`
	MaxIdleConns int `json:"max_idle_conns"`
}
