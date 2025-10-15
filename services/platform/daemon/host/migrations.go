package host

import (
	"errors"
	"os"
	"time"

	"github.com/steady-bytes/draft/pkg/chassis"
	"gopkg.in/yaml.v3"
)

type (
	Migrator interface {
		Migrate()
	}
	migrator struct {
		logger chassis.Logger
	}
	migrationsHistory struct {
		Migrations []migrationRun
	}
	migrationRun struct {
		Id        string
		Name      string
		Timestamp time.Time
		Error     string
	}
	migrationConfig struct {
		Id       string
		Name     string
		Required bool
		Run      func(logger chassis.Logger) error
	}
)

var (
	migrationsList = []migrationConfig{}
)

func NewMigrator(logger chassis.Logger) Migrator {
	return migrator{
		logger: logger,
	}
}

func (m migrator) Migrate() {
	m.logger.Info("running migrations")

	history := migrationsHistory{}
	// TODO: migrations should be stored in blueprint
	// f, err := os.ReadFile(MigrationsFile())
	var f []byte
	var err error
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.logger.Info("no migrations history file")
			history.Migrations = []migrationRun{}
		} else {
			m.logger.WithError(err).Panic("failed to open migrations history file")
		}
	} else {
		err = yaml.Unmarshal(f, &history)
		if err != nil {
			m.logger.WithError(err).Panic("failed to unmarshal migrations history file")
		}
	}

	for _, l := range migrationsList {
		complete := false
		for _, h := range history.Migrations {
			if l.Id == h.Id {
				complete = true
				break
			}
		}
		if !complete {
			log := m.logger.WithFields(
				chassis.Fields{
					"id":   l.Id,
					"name": l.Name,
				},
			)
			r := migrationRun{
				Id:        l.Id,
				Name:      l.Name,
				Timestamp: time.Now(),
			}
			log.Info("running migration")
			err := l.Run(m.logger)
			if err != nil {
				// TODO: send error to server
				log.WithError(err).Error("failed to run migration")
				if l.Required {
					m.logger.Panic("failed to run required migration")
				}
				r.Error = err.Error()
			}
			history.Migrations = append(history.Migrations, r)
		}
	}

	// data, err := yaml.Marshal(history)
	// if err != nil {
	// 	m.logger.WithError(err).Panic("failed to marshal migrations history")
	// }

	// TODO: migrations should be stored in blueprint
	// err = os.WriteFile(MigrationsFile(), data, 0666)
	if err != nil {
		m.logger.WithError(err).Panic("failed to write migrations history file")
	}

	m.logger.Info("migrations completed")
}
