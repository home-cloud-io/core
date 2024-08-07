package controller

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/home-cloud-io/core/services/platform/operator/internal/controller/secrets"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		dsn := fmt.Sprintf("postgres://postgres:%s@postgres.postgres.svc.cluster.local:5432/postgres?sslmode=disable", secret.Data["password"])
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		// create user
		password := secrets.Generate(24, true)
		_, err := db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", d.Name, password))
		if err != nil {
			return err
		}

		// create database
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s OWNER %s", d.Name, d.Name))
		if err != nil {
			return err
		}

		// execute init script (if provided)
		if len(d.Init) > 0 {
			// create db client (for new database)
			dsn := fmt.Sprintf("postgres://postgres:%s@postgres.postgres.svc.cluster.local:5432/%s?sslmode=disable", secret.Data["password"], d.Name)
			sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
			db := bun.NewDB(sqldb, pgdialect.New())
			_, err := db.Exec(d.Init)
			if err != nil {
				return err
			}
		}

		// create kube secret with access credentials
		err = r.Client.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", d.Type, d.Name),
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"hostname": []byte("postgres.postgres.svc.cluster.local"),
				"database": []byte(d.Name),
				"username": []byte(d.Name),
				"password": []byte(password),
				"port":     []byte("5432"),
				"uri":      []byte(fmt.Sprintf("postgres://%s:%s@postgres.postgres.svc.cluster.local:5432/%s?sslmode=disable", d.Name, password, d.Name)),
			},
		})
	case "mysql":
		// TODO
	default:
		return fmt.Errorf("unsupported database type requested: %s", d.Type)
	}

	return nil
}
