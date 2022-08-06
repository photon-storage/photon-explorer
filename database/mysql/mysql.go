package mysql

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

// NewMySQLDB create the mysql master/slaves cluster
func NewMySQLDB(cfg Config) (*gorm.DB, error) {
	dsnTemplate := "%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local"
	masterDSN := fmt.Sprintf(dsnTemplate,
		cfg.Master.Username,
		cfg.Master.Password,
		cfg.Master.Host,
		cfg.Master.Port,
		cfg.Master.DBName,
	)
	var slaveDSNs []gorm.Dialector
	for _, slave := range cfg.Slaves {
		slaveDSN := fmt.Sprintf(dsnTemplate,
			slave.Username,
			slave.Password,
			slave.Host,
			slave.Port,
			slave.DBName,
		)
		slaveDSNs = append(slaveDSNs, mysql.Open(slaveDSN))
	}

	db, err := gorm.Open(mysql.Open(masterDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(cfg.LogLevel)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "open master mysql")
	}

	dbResolverCfg := dbresolver.Config{
		Sources:  []gorm.Dialector{mysql.Open(masterDSN)},
		Replicas: slaveDSNs,
		Policy:   dbresolver.RandomPolicy{}}
	if err := db.Use(dbresolver.Register(dbResolverCfg).
		SetConnMaxIdleTime(time.Hour).
		SetConnMaxLifetime(24 * time.Hour).
		SetMaxIdleConns(cfg.ConnCfg.MaxIdleConns).
		SetMaxOpenConns(cfg.ConnCfg.MaxOpenConns),
	); err != nil {
		return nil, err
	}

	return db, nil
}
