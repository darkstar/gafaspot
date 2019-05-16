// Copyright 2019, Advanced UniByte GmbH.
// Author Marie Lohbeck.
//
// This file is part of Gafaspot.
//
// Gafaspot is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gafaspot is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gafaspot.  If not, see <https://www.gnu.org/licenses/>.

package database

import (
	"database/sql"
	"os"
	"time"

	logging "github.com/alexcesaro/log"

	// Gafaspot uses a SQLite database. Therefore, the go-sqlite3 package is used, as it is a
	// database driver for SQLite for Go's database/sql package
	_ "github.com/mattn/go-sqlite3"
	"github.com/AdvUni/gafaspot/util"
)

var (
	// ttlMonths is the general TTL for old database entries in months. Applies to tables users and reservations.
	ttlMonths int

	maxBookingDays   int
	maxQueuingMonths int

	db     *sql.DB
	logger logging.Logger
)

// InitDB prepares the database for gafaspot. Opens the database at the path given in config file.
// As SQLite is used, database doesn't even need to exist yet. Prepares all database tables and
// fills the environments table with the information from config file.
// Sets the package variable db to enable every function in the package to access the database.
func InitDB(l logging.Logger, config util.GafaspotConfig) {
	logger = l
	logger.Debugf("Database path is: %v", config.Database)

	// take over some constant values and set them as global vars
	ttlMonths = config.DBTTLmonths
	maxBookingDays = config.MaxBookingDays
	maxQueuingMonths = config.MaxQueuingMonths

	// Open database. SQLite databases are simple files, and if database doesn't exist yet, a new file will be created at the specified path
	var err error
	db, err = sql.Open("sqlite3", config.Database)
	if err != nil {
		logger.Emergency("Not able to open database: ", err)
		os.Exit(1)
	}

	// Create table reservations. If it already exists, don't overwrite
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS reservations (id INTEGER PRIMARY KEY, status TEXT NOT NULL, username TEXT NOT NULL, env_plain_name TEXT NOT NULL, start DATETIME NOT NULL, end DATETIME NOT NULL, subject TEXT, labels TEXT, delete_on DATE NOT NULL);")
	if err != nil {
		logger.Emergency(err)
		os.Exit(1)
	}

	// Create table users. If it already exists, don't overwrite
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (username TEXT UNIQUE NOT NULL, ssh_pub_key BLOB, delete_on DATE NOT NULL);")
	if err != nil {
		logger.Emergency(err)
		os.Exit(1)
	}

	// Create table environments. If it already exist, delete it first. Someone might have updated the environment configurations before system restart. So this table should be created from scratch.
	_, err = db.Exec("DROP TABLE IF EXISTS environments;")
	if err != nil {
		logger.Emergency(err)
		os.Exit(1)
	}
	_, err = db.Exec("CREATE TABLE environments (env_plain_name TEXT UNIQUE NOT NULL, env_nice_name TEXT NOT NULL, has_ssh BOOLEAN NOT NULL, description TEXT);")
	if err != nil {
		logger.Emergency(err)
		os.Exit(1)
	}

	// Fill empty table environments with information from configuration file
	for envPlainName, envConf := range config.Environments {
		envPlainName = util.CreatePlainIdentifier(envPlainName)
		envNiceName := envConf.NiceName
		if envNiceName == "" {
			envNiceName = envPlainName
		}
		envDescription := envConf.Description
		envHasSSH := false
		for _, secEng := range envConf.SecretsEngines {
			if secEng.EngineType == "ssh" {
				envHasSSH = true
			}
		}
		_, err = db.Exec("INSERT INTO environments VALUES (?, ?, ?, ?);", envPlainName, envNiceName, envHasSSH, envDescription)
		if err != nil {
			logger.Emergency(err)
			os.Exit(1)
		}
	}
}

func beginTransaction() *sql.Tx {
	tx, err := db.Begin()
	if err != nil {
		logger.Emergency(err)
		os.Exit(1)
	}
	return tx
}

func commitTransaction(tx *sql.Tx) {
	err := tx.Commit()
	if err != nil {
		logger.Error(err)
	}
}

func addTTL(t time.Time) time.Time {
	return t.AddDate(0, ttlMonths, 0)
}
