package ui

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"gitlab-vs.informatik.uni-ulm.de/gafaspot/database"
	"gitlab-vs.informatik.uni-ulm.de/gafaspot/util"
	"gitlab-vs.informatik.uni-ulm.de/gafaspot/vault"
)

type envReservations struct {
	Env                  util.Environment
	ReservationsUpcoming []util.Reservation
	ReservationsActive   []util.Reservation
	ReservationsExpired  []util.Reservation
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	errormessage := readErrorCookie(w, r)
	infomessage := readInfoCookie(w, r)

	err := loginformTmpl.Execute(w, map[string]interface{}{"Error": errormessage, "Info": infomessage})
	if err != nil {
		log.Println(err)
	}
}

func sortReservations(reservations []util.Reservation) ([]util.Reservation, []util.Reservation, []util.Reservation) {
	// sort reservation list by start date
	sort.Slice(reservations, func(i, j int) bool {
		return reservations[i].Start.Before(reservations[j].Start)
	})
	// split list into three sub lists
	var upcoming, active, expired []util.Reservation
	for _, r := range reservations {
		switch r.Status {
		case "upcoming":
			upcoming = append(upcoming, r)
		case "active":
			active = append(active, r)
		case "expired":
			expired = append(expired, r)
		case "error":
			log.Println("reservation with status error is not displayed")
		}
	}
	return upcoming, active, expired
}

func mainPageHandler(w http.ResponseWriter, r *http.Request) {
	username, ok := verifyUser(w, r)
	if !ok {
		redirectNotAuthenticated(w, r)
		return
	}
	envReservationsList := []envReservations{}

	for _, env := range environments {
		upcoming, active, expired := sortReservations(database.GetEnvReservations(env.PlainName))
		envReservationsList = append(envReservationsList, envReservations{env, upcoming, active, expired})
	}

	err := mainviewTmpl.Execute(w, map[string]interface{}{"Username": username, "Envcontent": envReservationsList})
	if err != nil {
		log.Println(err)
	}
}

func personalPageHandler(w http.ResponseWriter, r *http.Request) {
	username, ok := verifyUser(w, r)
	if !ok {
		redirectNotAuthenticated(w, r)
		return
	}

	sshEntry, ok := database.GetUserSSH(username)
	if !ok {
		sshEntry = "no key yet"
	}

	upcoming, active, expired := sortReservations(database.GetUserReservations(username))
	err := personalviewTmpl.Execute(w, map[string]interface{}{"Username": username, "SSHkey": sshEntry, "ReservationsUpcoming": upcoming, "ReservationsActive": active, "ReservationsExpired": expired})
	if err != nil {
		log.Println(err)
	}
}

func credsPageHandler(w http.ResponseWriter, r *http.Request) {
	username, ok := verifyUser(w, r)
	if !ok {
		redirectNotAuthenticated(w, r)
		return
	}
	envNames := database.GetUserActiveReservationEnv(username)
	log.Println(envNames)

	for _, env := range envNames {
		creds, err := vault.ReadCredentials(env, vault.CreateVaultToken())
		if err != nil {
			log.Println(err)
		}
		fmt.Fprintf(w, "creds for environment '%v':\n%v\n", env, creds)
	}
}

func newreservationPageHandler(w http.ResponseWriter, r *http.Request) {
	username, ok := verifyUser(w, r)
	if !ok {
		redirectNotAuthenticated(w, r)
		return
	}

	errormessage := readErrorCookie(w, r)

	selectedEnvPlainName := mux.Vars(r)["env"]
	envHasSSH, ok := envHasSSHMap[selectedEnvPlainName]
	if !ok {
		fmt.Fprint(w, "environment in url does not exist")
		return
	}
	sshMissing := envHasSSH && !database.UserHasSSH(username)

	err := reservationformTmpl.Execute(w, map[string]interface{}{"Username": username, "Envs": environments, "Selected": selectedEnvPlainName, "SSHmissing": sshMissing, "Error": errormessage})
	if err != nil {
		log.Println(err)
	}
}

func reserveHandler(w http.ResponseWriter, r *http.Request) {
	username, ok := verifyUser(w, r)
	if !ok {
		redirectNotAuthenticated(w, r)
		return
	}
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}

	var reservation util.Reservation

	reservation.User = username

	reservation.EnvPlainName = template.HTMLEscapeString(r.Form.Get("env"))
	if reservation.EnvPlainName == "" {
		redirectInvalidReservation(w, r, "environment invalid")
		return
	}

	startstring := template.HTMLEscapeString(r.Form.Get("startdate")) + " " + template.HTMLEscapeString(r.Form.Get("starttime"))
	reservation.Start, err = time.ParseInLocation(util.TimeLayout, startstring, time.Local)
	if err != nil {
		log.Println(err)
		redirectInvalidReservation(w, r, "start date/time malformed")
		return
	}

	endstring := template.HTMLEscapeString(r.Form.Get("enddate")) + " " + template.HTMLEscapeString(r.Form.Get("endtime"))
	reservation.End, err = time.ParseInLocation(util.TimeLayout, endstring, time.Local)
	if err != nil {
		log.Println(err)
		redirectInvalidReservation(w, r, "end date/time malformed")
		return
	}

	reservation.Subject = template.HTMLEscapeString(r.Form.Get("sub"))

	err = database.CreateReservation(reservation)
	if err != nil {
		redirectInvalidReservation(w, r, err.Error())
		return
	}

	err = reservesuccessTmpl.Execute(w, reservation)
	if err != nil {
		log.Println(err)
	}
}
