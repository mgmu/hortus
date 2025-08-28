package database

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgmu/hortus/internal/plants"
	"log"
	"os"
)

type PostgresDatabase struct {
	pool *pgxpool.Pool
}

// Connect attempts to connect to the Postgres database and to set the
// appropriate schema for future queries.
func (db *PostgresDatabase) Connect() error {
	dburl := os.Getenv("HORTUS_DB_URL")
	if dburl == "" {
		return errors.New("database: Database URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dburl)
	if err != nil {
		return err
	}
	db.pool = pool

	if db.pool == nil {
		log.Fatal("db.pool is nil")
	}

	query := `
SELECT EXISTS (
SELECT FROM pg_tables
WHERE schemaname = 'hortus_schema'
AND (tablename = 'plant' OR tablename = 'plant_log')
);`
	var exist bool
	err = db.pool.QueryRow(context.Background(), query).Scan(&exist)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New("database: Schema and tables not found")
	}

	query = `SET search_path TO hortus_schema;`
	_, err = db.pool.Exec(context.Background(), query)
	if err != nil {
		return err
	}
	return nil
}

// Close closes all connections to this connection pool. Always returns a nil
// error.
func (db *PostgresDatabase) Close() error {
	db.pool.Close()
	return nil
}

// GetPlantsShortDescription queries the database for the identifier and common
// names of all plants in the 'plants' table and returns them in a slice.
func (db *PostgresDatabase) GetPlantsShortDescription() ([]plants.PlantShortDesc, error) {
	query := "SELECT id, common_name FROM plant;"
	rows, _ := db.pool.Query(context.Background(), query)
	plants, err := pgx.CollectRows(
		rows,
		pgx.RowToStructByPos[plants.PlantShortDesc],
	)
	if err != nil {
		return nil, err
	}
	return plants, nil
}

// AddNewPlant attempts to insert a new entry in the 'plants' table with the
// provided common, generic and specific names of the plant. On success, returns
// the identifier of the inserted entry and a nil error.
func (db *PostgresDatabase) AddNewPlant(comm, gen, spe string) (int, error) {
	row := db.pool.QueryRow(
		context.Background(),
		`
INSERT INTO plant (common_name, generic_name, specific_name)
VALUES ($1, $2, $3)
RETURNING id;`,
		comm,
		gen,
		spe,
	)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetPlantNames queries the database for the common, generic and specific names
// of the plant of given id.
func (db *PostgresDatabase) GetPlantNames(id int) (string, string, string, error) {
	row := db.pool.QueryRow(
		context.Background(),
		"SELECT * FROM plant WHERE id=$1;",
		id,
	)
	var comm, gen, spe string
	err := row.Scan(&id, &comm, &gen, &spe)
	if err != nil {
		return "", "", "", err
	}
	return comm, gen, spe, nil
}

// GetPlantsLog queries the database for all the logs of the plant of given
// identifier and returns them in a slice.
func (db *PostgresDatabase) GetPlantLogs(id int) ([]plants.PlantLog, error) {
	rows, _ := db.pool.Query(
		context.Background(),
		"SELECT * FROM plant_log WHERE plant_id=$1;",
		id,
	)
	plantLogs, err := pgx.CollectRows(
		rows,
		pgx.RowToStructByPos[plants.PlantLog],
	)
	if err != nil {
		return nil, err
	}
	return plantLogs, nil
}

// AddNewPlantLog attempts to insert a new entry in the 'plants_log` table for
// the plant of given identifier with given description and event type.
func (db *PostgresDatabase) AddNewPlantLog(id int, desc string, event int) error {
	// Insert new log entry
	row := db.pool.QueryRow(
		context.Background(),
		`
INSERT INTO plant_log (plant_id, description, event_type)
VALUES ($1, $2, $3)
RETURNING id;`,
		id,
		desc,
		event,
	)
	var logId int
	err := row.Scan(&logId)
	if err != nil {
		return err
	}
	return nil
}
