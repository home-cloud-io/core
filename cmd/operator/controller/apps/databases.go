package apps

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/home-cloud-io/core/cmd/operator/controller/secrets"
)

func (r *AppReconciler) createDatabase(ctx context.Context, d AppDatabase, namespace string) error {
	log.FromContext(ctx).Info("creating database", "db_name", d.Name, "db_type", d.Type)

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
		hostname := "postgres.postgres"
		if r.Config.Env() == "local" {
			hostname = "localhost"
		}
		dsn := fmt.Sprintf("postgres://postgres:%s@%s:5432/postgres?sslmode=disable", secret.Data["password"], hostname)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		err = r.createPostgresUser(ctx, db, d, namespace)
		if err != nil {
			return err
		}

		err = r.createPostgresUserDatabase(ctx, db, d, secret)
		if err != nil {
			return err
		}
	case "mysql":
		// TODO
	default:
		return fmt.Errorf("unsupported database type requested: %s", d.Type)
	}

	return nil
}

func (r *AppReconciler) deleteDatabase(ctx context.Context, d AppDatabase, namespace string) error {
	log.FromContext(ctx).Info("deleting database", "db_name", d.Name, "db_type", d.Type)

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
		hostname := "postgres.postgres"
		if r.Config.Env() == "local" {
			hostname = "localhost"
		}
		dsn := fmt.Sprintf("postgres://postgres:%s@%s:5432/postgres?sslmode=disable", secret.Data["password"], hostname)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		// delete user database
		err = r.deletePostgresUserDatabase(ctx, db, d)
		if err != nil {
			return err
		}

		// delete user
		err = r.deletePostgresUser(ctx, db, d, namespace)
		if err != nil {
			return err
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
	var (
		objName = fmt.Sprintf("%s-%s", d.Type, d.Name)
		pass    []byte
	)

	// check for existing secret
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      objName,
		Namespace: namespace,
	}, secret)
	if client.IgnoreNotFound(err) != nil {
		return err
	} else {
		var ok bool
		pass, ok = secret.Data["password"]
		if !ok {
			return fmt.Errorf("database secret contained no 'password' key")
		}
	}

	// create secret if not found
	if errors.IsNotFound(err) {
		pass, err = secrets.Generate(24, true)
		if err != nil {
			return err
		}

		// create kube secret with access credentials
		err = r.Client.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      objName,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				// TODO: consider making hostname:port configurable in setting so users can bring their own database
				"hostname": []byte("postgres.postgres"),
				"database": []byte(d.Name),
				"username": []byte(d.Name),
				"password": []byte(pass),
				"port":     []byte("5432"),
				"uri":      []byte(fmt.Sprintf("postgresql://%s:%s@postgres.postgres:5432/%s?sslmode=disable", d.Name, pass, d.Name)),
			},
		})
		if err != nil {
			return err
		}
	}

	// check if user already exists
	exists, err := sysObjectExists(ctx, db, fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", d.Name))
	if err != nil {
		return err
	}

	// update if exists, else create
	if exists {
		_, err = db.ExecContext(ctx, fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'", d.Name, pass))
	} else {
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", d.Name, pass))
	}

	return err
}

func (r *AppReconciler) deletePostgresUser(ctx context.Context, db *bun.DB, d AppDatabase, namespace string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf("DROP USER IF EXISTS %s", d.Name))
	if err != nil {
		return err
	}

	// delete kube secret with access credentials
	err = r.Client.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", d.Type, d.Name),
			Namespace: namespace,
		},
	})
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

func (r *AppReconciler) createPostgresUserDatabase(ctx context.Context, db *bun.DB, d AppDatabase, secret *corev1.Secret) error {
	// check if user database already exists (this happens on a reinstall without wiping old data)
	exists, err := sysObjectExists(ctx, db, fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", d.Name))
	if err != nil {
		return err
	}
	if !exists {
		// create database for user (using system db client)
		_, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s OWNER %s", d.Name, d.Name))
		if err != nil {
			return err
		}
	}

	// execute init script (if provided)
	if len(d.Init) > 0 {
		// create db client (for user database)
		hostname := "postgres.postgres"
		if r.Config.Env() == "local" {
			hostname = "localhost"
		}
		dsn := fmt.Sprintf("postgres://postgres:%s@%s:5432/%s?sslmode=disable", secret.Data["password"], hostname, d.Name)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())
		_, err := db.ExecContext(ctx, d.Init)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *AppReconciler) deletePostgresUserDatabase(ctx context.Context, db *bun.DB, d AppDatabase) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", d.Name))
	if err != nil {
		return err
	}
	return nil
}
