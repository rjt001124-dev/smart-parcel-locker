package data

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
)

const dependencyProbeTimeout = 5 * time.Second

type resourceCloser interface {
	Close() error
}

type Clients struct {
	MySQL *sql.DB
	Redis *redis.Client

	mysqlCloser resourceCloser
	redisCloser resourceCloser
	closeOnce   sync.Once
	closeErr    error
}

func MySQLDSN(cfg conf.MySQLConfig) string {
	mysqlConfig := mysql.NewConfig()
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	mysqlConfig.DBName = cfg.Database
	mysqlConfig.User = cfg.User
	mysqlConfig.Passwd = cfg.Password
	mysqlConfig.ParseTime = true
	mysqlConfig.Loc = time.UTC
	mysqlConfig.MultiStatements = false
	return mysqlConfig.FormatDSN()
}

func RedisOptions(cfg conf.RedisConfig) *redis.Options {
	return &redis.Options{
		Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.Database,
	}
}

func Open(ctx context.Context, cfg conf.Config) (*Clients, error) {
	mysqlClient, err := sql.Open("mysql", MySQLDSN(cfg.MySQL))
	if err != nil {
		return nil, errors.New("mysql dependency unavailable")
	}
	mysqlClient.SetMaxOpenConns(20)
	mysqlClient.SetMaxIdleConns(10)
	mysqlClient.SetConnMaxLifetime(30 * time.Minute)
	mysqlClient.SetConnMaxIdleTime(5 * time.Minute)

	redisClient := redis.NewClient(RedisOptions(cfg.Redis))
	clients := &Clients{
		MySQL:       mysqlClient,
		Redis:       redisClient,
		mysqlCloser: mysqlClient,
		redisCloser: redisClient,
	}

	probeCtx, cancel := context.WithTimeout(ctx, dependencyProbeTimeout)
	defer cancel()
	if err := mysqlClient.PingContext(probeCtx); err != nil {
		_ = clients.Close()
		return nil, errors.New("mysql dependency unavailable")
	}
	_ = redisClient.Ping(probeCtx).Err()

	return clients, nil
}

func (c *Clients) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		var closeErrors []error
		if c.mysqlCloser != nil {
			if err := c.mysqlCloser.Close(); err != nil {
				closeErrors = append(closeErrors, errors.New("mysql close failed"))
			}
		}
		if c.redisCloser != nil {
			if err := c.redisCloser.Close(); err != nil {
				closeErrors = append(closeErrors, errors.New("redis close failed"))
			}
		}
		c.closeErr = errors.Join(closeErrors...)
	})
	return c.closeErr
}

func newClientsWithClosers(mysqlCloser, redisCloser resourceCloser) *Clients {
	return &Clients{mysqlCloser: mysqlCloser, redisCloser: redisCloser}
}
