package data

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
)

func TestMySQLDSN(t *testing.T) {
	dsn := MySQLDSN(conf.MySQLConfig{
		Host:     "mysql",
		Port:     3306,
		Database: "locker",
		User:     "app",
		Password: "secret",
	})

	got, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("MySQLDSN returned an invalid DSN")
	}
	if got.Net != "tcp" {
		t.Errorf("Net = %q, want tcp", got.Net)
	}
	if got.Addr != "mysql:3306" {
		t.Errorf("Addr = %q, want mysql:3306", got.Addr)
	}
	if got.DBName != "locker" {
		t.Errorf("DBName = %q, want locker", got.DBName)
	}
	if got.User != "app" {
		t.Errorf("User = %q, want app", got.User)
	}
	if got.Passwd != "secret" {
		t.Error("password was not preserved")
	}
	if !got.ParseTime {
		t.Error("ParseTime = false, want true")
	}
	if got.Loc != time.UTC {
		t.Errorf("Loc = %v, want UTC", got.Loc)
	}
	if got.MultiStatements {
		t.Error("MultiStatements = true, want false")
	}
}

func TestClientsCloseIsSafeForConcurrentCallers(t *testing.T) {
	mysqlCloser := &countingCloser{err: errors.New("mysql raw close error")}
	redisCloser := &countingCloser{err: errors.New("redis raw close error")}
	clients := newClientsWithClosers(mysqlCloser, redisCloser)

	const callers = 20
	errs := make([]error, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := range callers {
		go func() {
			defer wg.Done()
			errs[i] = clients.Close()
		}()
	}
	wg.Wait()

	if mysqlCloser.calls != 1 {
		t.Errorf("MySQL close calls = %d, want 1", mysqlCloser.calls)
	}
	if redisCloser.calls != 1 {
		t.Errorf("Redis close calls = %d, want 1", redisCloser.calls)
	}
	for i, err := range errs {
		if err != errs[0] {
			t.Errorf("Close caller %d returned a different error instance", i)
		}
		if err == nil || err.Error() != "mysql close failed\nredis close failed" {
			t.Errorf("Close caller %d error = %v, want joined sanitized errors", i, err)
		}
	}
}

func TestRedisOptions(t *testing.T) {
	got := RedisOptions(conf.RedisConfig{
		Host:     "redis",
		Port:     6379,
		Username: "app",
		Password: "secret",
		Database: 3,
	})

	if got.Addr != "redis:6379" {
		t.Errorf("Addr = %q, want redis:6379", got.Addr)
	}
	if got.Username != "app" {
		t.Errorf("Username = %q, want app", got.Username)
	}
	if got.Password != "secret" {
		t.Error("password was not preserved")
	}
	if got.DB != 3 {
		t.Errorf("DB = %d, want 3", got.DB)
	}
}

func TestClientsCloseIsIdempotentAndClosesEveryResource(t *testing.T) {
	mysqlCloser := &countingCloser{err: errors.New("mysql raw close error")}
	redisCloser := &countingCloser{err: errors.New("redis raw close error")}
	clients := newClientsWithClosers(mysqlCloser, redisCloser)

	err1 := clients.Close()
	err2 := clients.Close()

	if mysqlCloser.calls != 1 {
		t.Errorf("MySQL close calls = %d, want 1", mysqlCloser.calls)
	}
	if redisCloser.calls != 1 {
		t.Errorf("Redis close calls = %d, want 1", redisCloser.calls)
	}
	if err1 == nil {
		t.Fatal("first Close error = nil, want sanitized errors")
	}
	if err2 == nil || err2.Error() != err1.Error() {
		t.Fatal("repeated Close did not return the same result")
	}
	if err1.Error() != "mysql close failed\nredis close failed" {
		t.Fatalf("Close error = %q, want joined sanitized errors", err1)
	}
}

type countingCloser struct {
	calls int
	err   error
}

func (c *countingCloser) Close() error {
	c.calls++
	return c.err
}
