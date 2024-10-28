package host

import (
	"errors"
	"os"
	"strings"
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
	migrationsList = []migrationConfig{
		{
			Id:       "60fbc7c6-9388-4c33-9a02-4da87de5ba6d",
			Name:     "Grant server read permissions on all cluster resources",
			Run:      m1,
			Required: true,
		},
	}
)

func NewMigrator(logger chassis.Logger) Migrator {
	return migrator{
		logger: logger,
	}
}

func (m migrator) Migrate() {
	m.logger.Info("running migrations")

	history := migrationsHistory{}
	f, err := os.ReadFile(MigrationsFile)
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
			r := migrationRun{
				Id:        l.Id,
				Name:      l.Name,
				Timestamp: time.Now(),
			}
			err := l.Run(m.logger)
			if err != nil {
				// TODO: send error to server
				m.logger.WithError(err).WithFields(
					chassis.Fields{
						"id":   l.Id,
						"name": l.Name,
					},
				).Error("failed to run migration")
				if l.Required {
					m.logger.Panic("failed to run required migration")
				}
				r.Error = err.Error()
			}
			history.Migrations = append(history.Migrations, r)
		}
	}

	data, err := yaml.Marshal(history)
	if err != nil {
		m.logger.WithError(err).Panic("failed to marshal migrations history")
	}

	err = os.WriteFile(MigrationsFile, data, 0666)
	if err != nil {
		m.logger.WithError(err).Panic("failed to write migrations history file")
	}

	m.logger.Info("migrations completed")
}

func m1(logger chassis.Logger) error {
	var (
		replacers = []Replacer{}
		fileName  = ServerManifestFile
	)

	replacers = append(replacers, func(line string) string {
		if line == "  - pods" {
			line = "  - \"*\""
		}
		return line
	})
	replacers = append(replacers, func(line string) string {
		if strings.Contains(line, "read-pods") {
			line = strings.ReplaceAll(line, "read-pods", "read-all")
		}
		return line
	})

	err := LineByLineReplace(fileName, replacers)
	if err != nil {
		return err
	}

	return nil
}
