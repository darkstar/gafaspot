package ui

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"gitlab-vs.informatik.uni-ulm.de/gafaspot/constants"
)

const (
	yearsTTL = 2
)

type ReservationError string

func (err ReservationError) Error() string {
	return fmt.Sprintf("reservation is invalid: %v", string(err))
}

func CreateReservation(db *sql.DB, username, envName, subject, labels string, start, end time.Time) error {

	// check, whether reservation is in future
	if !start.After(time.Now()) {
		return ReservationError("cannot do reservation for the past")
	}

	// check whether start < end
	if !start.Before(end) {
		return ReservationError("end of reservation must be after start of reservation")
	}

	// start a transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	// defer the transaction's commit as this function may return an error
	defer func() {
		err = tx.Commit()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// check, whether environment exists and determine, whether the reservation needs an ssh key
	stmt, err := tx.Prepare("SELECT has_ssh FROM environments WHERE (env_name=?);")
	defer stmt.Close()
	if err != nil {
		log.Println(err)
	}

	var hasSSH bool
	err = stmt.QueryRow(envName).Scan(&hasSSH)
	if err == sql.ErrNoRows {
		return ReservationError(fmt.Sprintf("environment %v does not exist", envName))
	} else if err != nil {
		log.Fatal(err)
	}

	// check, whether there is stored an ssh key for the user, if it is needed for the reservation
	if hasSSH {
		var sshKey sql.NullString
		stmt, err = tx.Prepare("SELECT ssh_pub_key FROM users WHERE (username=?);")
		defer stmt.Close()
		if err != nil {
			log.Println(err)
		}
		err = stmt.QueryRow(username).Scan(&sshKey)
		if err == sql.ErrNoRows || !sshKey.Valid {
			return ReservationError(fmt.Sprintf("there is no ssh public key stored for user %v, but it is required for booking environment %v", username, envName))
		} else if err != nil {
			log.Println(err)
		}
	}

	// check the environment's availability within the requested time range:
	// a conflict occurs iff ((start1 <= end2) && (end1 >= start2))
	stmt, err = tx.Prepare("SELECT start, end FROM reservations WHERE (env_name=?) AND (start<=?) AND (end>=?);")
	defer stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	var conflictStart, conflictEnd time.Time
	err = stmt.QueryRow(envName, end, start).Scan(&conflictStart, &conflictEnd)
	// there is a conflict, if answer is NOT empty; means, if there is NO sql.ErrNoRows
	if err == nil {
		return ReservationError(fmt.Sprintf("reservation conflicts with an existing reservation from %v to %v", conflictStart.Format(constants.TimeLayout), conflictEnd.Format(constants.TimeLayout)))
	}
	if err != sql.ErrNoRows {
		log.Fatal(err)
	}

	// generate the deletion date of reservation entry in database
	reservationDeleteDate := end.AddDate(yearsTTL, 0, 0)

	// finally write reservation into database
	stmt, err = tx.Prepare("INSERT INTO reservations (status, username, env_name, start, end, subject, labels, delete_on) VALUES(?,?,?,?,?,?,?,?);")
	defer stmt.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec("upcoming", username, envName, start, end, subject, labels, reservationDeleteDate)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func AbortReservation(db *sql.DB, username, string, id int) error {
	// start a transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	// defer the transaction's commit as this function may return an error
	defer func() {
		err = tx.Commit()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// fetch reservation from database
	stmt, err := tx.Prepare("SELECT status FROM reservations WHERE (username=?) AND (id=?);")
	defer stmt.Close()
	if err != nil {
		log.Fatal(err)
	}
	status := ""
	err = stmt.QueryRow(username, id).Scan(&status)
	if err == sql.ErrNoRows {
		log.Println(fmt.Errorf("tried to abort reservation which does not exist or not belongs to specified user; id '%v', user '%v'", id, username))
		return nil
	}
	if err != nil {
		log.Fatal(err)
	}

	// check reservation status (can only abort upcoming reservations)
	if status != "upcoming" {
		return fmt.Errorf("reservation is already active or expired, though it is not possible anymore to abort it")
	}

	// delete reservation from database
	stmt, err = tx.Prepare("DELETE FROM reservations WHERE id=?;")
	defer stmt.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(id)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
