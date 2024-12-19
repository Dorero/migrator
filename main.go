package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"strings"
	"time"

	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		DbName   string `yaml:"db_name"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Type     string `yaml:"type"`
	} `yaml:"database"`
}

func initStructure() {
	_, err := os.Stat("db")

	if os.IsNotExist(err) {
		err = os.Mkdir("db", 0755)
		if err != nil {
			log.Fatal(err)
		}

		err = os.Mkdir("db/migrations", 0755)
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Create("db/schema.sql")

		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()

		content := "CREATE TABLE schema ( \n\tid SERIAL PRIMARY KEY,\n\tname VARCHAR(255) UNIQUE NOT NULL,\n\tapplied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL\n);"

		_, err = file.WriteString(content)
		if err != nil {
			log.Fatal(err)
		}

		config := Config{
			Database: struct {
				DbName   string `yaml:"db_name"`
				User     string `yaml:"user"`
				Password string `yaml:"password"`
				Host     string `yaml:"host"`
				Port     int    `yaml:"port"`
				Type     string `yaml:"type"`
			}{
				DbName:   "postgres",
				User:     "postgres",
				Password: "postgres",
				Host:     "localhost",
				Port:     5432,
				Type:     "postgresql",
			},
		}

		file, err = os.Create("db/config.yaml")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		encoder := yaml.NewEncoder(file)
		err = encoder.Encode(&config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Initialization completed successfully")
	} else if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Initialization has already been done")
	}
}

func main() {
	args := os.Args

	if len(args) == 1 {
		help()
	} else {
		switch args[1] {
		case "init":
			initStructure()
		case "create":
			if len(args) == 2 {
				fmt.Println("It is necessary to pass the table name")
			} else {
				resource := args[2]
				file, err := os.Create(fmt.Sprintf("db/migrations/%d_%s.sql", time.Now().UnixMilli(), resource))

				if err != nil {
					log.Fatal(err)
				}

				defer file.Close()

				content := fmt.Sprintf("-- Migration: up\nCREATE TABLE %s ( \n\tid SERIAL PRIMARY KEY,\n\tcreated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,\n\tupdated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n);\n-- Migration: down\nDROP TABLE IF EXISTS %s;", resource, resource)

				_, err = file.WriteString(content)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Printf("%s table successfully created\n", resource)
			}

		case "migrate":
			files, err := os.ReadDir("db/migrations")
			if err != nil {
				log.Fatal(err)
			}

			if len(files) == 0 {
				log.Fatal("No migrations found")
			}

			var fileMigrations []string
			for _, file := range files {
				fileMigrations = append(fileMigrations, file.Name())
			}

			config := dbConfig()
			connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable", config.Database.User, config.Database.Password, config.Database.DbName, config.Database.Host, config.Database.Port)

			db, err := sql.Open("postgres", connStr)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			err = db.Ping()
			if err != nil {
				log.Fatal(err)
			}

			var schemaCreated sql.NullString

			if config.Database.Type == "postgresql" {
				err = db.QueryRow("SELECT to_regclass('public.schema');").Scan(&schemaCreated)
				if err != nil {
					log.Fatal(err)
				}

			}

			if !schemaCreated.Valid {
				content, err := os.ReadFile("db/schema.sql")
				if err != nil {
					log.Fatal(err)
				}

				_, err = db.Exec(string(content))
				if err != nil {
					log.Fatal(err)
				}
			}

			rows, err := db.Query("SELECT name from schema;")
			if err != nil {
				log.Fatal(err)
			}
			defer rows.Close()

			var appliedMigrations []string

			for rows.Next() {
				var name string
				err := rows.Scan(&name)
				if err != nil {
					log.Fatal(err)
				}

				appliedMigrations = append(appliedMigrations, name)
			}

			if err := rows.Err(); err != nil {
				log.Fatal(err)
			}

			migrations := subtractSlices(fileMigrations, appliedMigrations)

			if len(migrations) == 0 {
				fmt.Println("No new migrations")
			} else {
				for _, migration := range migrations {
					content, err := os.ReadFile(fmt.Sprintf("db/migrations/%s", migration))
					if err != nil {
						log.Fatal(err)
					}

					mig, err := extractMigrationScripts(string(content))
					if err != nil {
						log.Fatal(err)
					}

					_, err = db.Exec(string(mig["up"]))
					if err != nil {
						log.Fatal(err)
					}

					_, err = db.Exec(fmt.Sprintf("insert into schema (name) values ('%s');", string(migration)))
					if err != nil {
						log.Fatal(err)
					}
				}

				fmt.Println("Migration completed successfully")
			}

		case "rollback":
			config := dbConfig()
			connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable", config.Database.User, config.Database.Password, config.Database.DbName, config.Database.Host, config.Database.Port)

			db, err := sql.Open("postgres", connStr)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			err = db.Ping()
			if err != nil {
				log.Fatal(err)
			}

			var rows *sql.Rows

			if len(args) == 3 {
				if args[2] == "all" {
					rows, err = db.Query("SELECT name from schema order by applied_at desc;")
				} else {
					log.Fatal("Unsupported command")
				}
			} else {
				rows, err = db.Query("SELECT name from schema order by applied_at desc limit 1;")
			}

			if err != nil {
				log.Fatal(err)
			}
			defer rows.Close()

			var appliedMigrations []string

			for rows.Next() {
				var name string
				err := rows.Scan(&name)
				if err != nil {
					log.Fatal(err)
				}

				appliedMigrations = append(appliedMigrations, name)
			}

			if err := rows.Err(); err != nil {
				log.Fatal(err)
			}

			if len(appliedMigrations) == 0 {
				fmt.Println("No migrations applied")
			} else {
				for _, migration := range appliedMigrations {
					content, err := os.ReadFile(fmt.Sprintf("db/migrations/%s", migration))
					if err != nil {
						log.Fatal(err)
					}

					mig, err := extractMigrationScripts(string(content))
					if err != nil {
						log.Fatal(err)
					}

					_, err = db.Exec(string(mig["down"]))
					if err != nil {
						log.Fatal(err)
					}

					_, err = db.Exec(fmt.Sprintf("delete from schema where name = '%s';", string(migration)))
					if err != nil {
						log.Fatal(err)
					}
				}

				if len(args) == 3 {
					if args[2] == "all" {
						fmt.Println("Rollback of all migration completed successfully")
					}
				} else {
					fmt.Println("Rollback completed successfully")
				}

			}
		case "help":
			help()
		default:
			help()
		}
	}

}

func help() {
	fmt.Println("Migrator is a tool for working with migrations")
	fmt.Println("Usage: migrator <command>")
	fmt.Println("The commands are:")
	fmt.Println("\t init - creates a db folder, a migrations folder, a sql schema for storing applied migrations, and a config for connecting to the database.")
	fmt.Println("\t create - creates a file with the sql extension")
	fmt.Println("\t migrate - applies all unapplied migrations")
	fmt.Println("\t rollback - rolls back the last migration")
	fmt.Println("\t rollback all - rolls back all migrations")
	fmt.Println("\t help - print information")
}

func dbConfig() Config {
	file, err := os.Open("db/config.yaml")
	if err != nil {
		panic("Error opening file ")
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		panic("YAML reading error")
	} else {
		return config
	}
}

func subtractSlices(slice1, slice2 []string) []string {
	set := make(map[string]struct{})
	for _, v := range slice2 {
		set[v] = struct{}{}
	}

	var result []string
	for _, v := range slice1 {
		if _, found := set[v]; !found {
			result = append(result, v)
		}
	}

	return result
}

func extractMigrationScripts(fileContent string) (map[string]string, error) {
	migrations := make(map[string]string)

	upStart := strings.Index(fileContent, "-- Migration: up")
	if upStart == -1 {
		return nil, fmt.Errorf("missing 'up' migration")
	}
	upEnd := strings.Index(fileContent[upStart:], "-- Migration: down")
	if upEnd == -1 {
		return nil, fmt.Errorf("missing 'down' migration")
	}
	migrations["up"] = strings.TrimSpace(fileContent[upStart+len("-- Migration: up") : upStart+upEnd])

	downStart := strings.Index(fileContent, "-- Migration: down")
	if downStart == -1 {
		return nil, fmt.Errorf("missing 'down' migration")
	}
	migrations["down"] = strings.TrimSpace(fileContent[downStart+len("-- Migration: down"):])

	return migrations, nil
}
