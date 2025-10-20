package controller

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/home-cloud-io/core/services/platform/operator/internal/controller/secrets"
)

const (
	PostgresHostname = "postgres.postgres"
	// PostgresHostname = "localhost" // for local dev
)

func (r *AppReconciler) createDatabase(ctx context.Context, d AppDatabase, namespace string) error {

	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: d.Type,
		Name:      d.Type,
	}, secret)
	if err != nil {
		return fmt.Errorf("failed to get database secret: %s", err.Error())
	}

	switch d.Type {
	case "postgres":
		// create db client
		dsn := fmt.Sprintf("postgres://postgres:%s@%s:5432/postgres?sslmode=disable", secret.Data["password"], PostgresHostname)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		// check if user already exists (this happens on a reinstall without wiping old data)
		exists, err := sysObjectExists(ctx, db, fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", d.Name))
		if err != nil {
			return err
		}
		if !exists {
			err = r.createPostgresUser(ctx, db, d, namespace)
			if err != nil {
				return err
			}
		}

		// check if user database already exists (this happens on a reinstall without wiping old data)
		exists, err = sysObjectExists(ctx, db, fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", d.Name))
		if err != nil {
			return err
		}
		if !exists {
			err = createPostgresUserDatabase(ctx, db, d, secret)
			if err != nil {
				return err
			}
		}
	case "mysql":
		// TODO
	default:
		return fmt.Errorf("unsupported database type requested: %s", d.Type)
	}

	return nil
}

func sysObjectExists(ctx context.Context, db *bun.DB, query string) (bool, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return false, err
		}
	}
	if count != 1 {
		return false, nil
	}
	return true, nil
}

func (r *AppReconciler) createPostgresUser(ctx context.Context, db *bun.DB, d AppDatabase, namespace string) error {
	// create user within database
	pass, err := secrets.Generate(24, true)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", d.Name, pass))
	if err != nil {
		return err
	}

	// create kube secret with access credentials
	err = r.Client.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", d.Type, d.Name),
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"hostname": []byte("postgres.postgres"),
			"database": []byte(d.Name),
			"username": []byte(d.Name),
			"password": []byte(pass),
			"port":     []byte("5432"),
			"uri":      []byte(fmt.Sprintf("postgres://%s:%s@postgres.postgres:5432/%s?sslmode=disable", d.Name, pass, d.Name)),
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	return nil
}

func createPostgresUserDatabase(ctx context.Context, db *bun.DB, d AppDatabase, secret *corev1.Secret) error {
	// create database for user (using system db client)
	_, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s OWNER %s", d.Name, d.Name))
	if err != nil {
		return err
	}

	// execute init script (if provided)
	if len(d.Init) > 0 {
		// create db client (for user database)
		dsn := fmt.Sprintf("postgres://postgres:%s@%s:5432/%s?sslmode=disable", secret.Data["password"], PostgresHostname, d.Name)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())
		_, err := db.ExecContext(ctx, d.Init)
		if err != nil {
			return err
		}
	}

	return nil
}
