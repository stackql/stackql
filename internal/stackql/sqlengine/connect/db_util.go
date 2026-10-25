package db_util //nolint:revive,stylecheck // decent package name

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/stackql/stackql/internal/stackql/logging"
)

func Open(driverName, dbName, dsn string, maxRetries, retryDelaySeconds int) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("cannot init %s TCP connection with empty dsn", dbName)
	}
	db, err := sql.Open(driverName, dsn)
	retryCount := 0
	for {
		if retryCount >= maxRetries || err == nil {
			break
		}
		time.Sleep(time.Duration(retryDelaySeconds) * time.Second)
		db, err = sql.Open(driverName, dsn)
		retryCount++
	}
	if err != nil {
		return nil, fmt.Errorf("%s db object setup error = '%w'", driverName, err)
	}
	logging.GetLogger().Debugln(
		fmt.Sprintf("opened %s TCP db with connection string = '%s' and err  = '%v'",
			dbName, dsn, err))
	pingErr := db.Ping()
	retryCount = 0
	for {
		if retryCount >= maxRetries || pingErr == nil {
			break
		}
		time.Sleep(time.Duration(retryDelaySeconds) * time.Second)
		pingErr = db.Ping()
		retryCount++
	}
	if pingErr != nil {
		return nil, fmt.Errorf("%s connection setup ping error = '%w'", dbName, pingErr)
	}
	logging.GetLogger().Debugln(
		fmt.Sprintf(
			"opened and pinged %s TCP db with connection string = '%s' and err  = '%v'",
			dbName, dsn, err))
	return db, nil
}
