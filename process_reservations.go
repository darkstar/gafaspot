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

package main

import (
	"time"

	logging "github.com/alexcesaro/log"
	"github.com/AdvUni/gafaspot/database"
	"github.com/AdvUni/gafaspot/vault"
)

const (
	scanningInterval = 5 * time.Minute
)

var (
	logger logging.Logger
)

// handleReservationScanning is an endless loop which repetitively calls reservationScan. This is
// keeping the reservations table in database up-to-date.
func handleReservationScanning(l logging.Logger) {
	logger = l

	// endless loop, triggered each 5 minutes
	tick := time.NewTicker(scanningInterval).C
	for {
		<-tick
		reservationScan()
		logger.Info("executed reservation scan")
	}
}

// reservationScan reads through all reservations in database and checks if there must be performed
// an action as starting or ending them.
func reservationScan() {

	now := time.Now()

	// any active bookings which should end?
	database.ExpireActiveReservations(now, vault.EndBooking)

	// any upcoming bookings which should start?
	database.StartUpcomingReservations(now, vault.StartBooking)

	// any expired bookings which should get deleted?
	database.DeleteOldReservations(now)

	// finally, check if some of the entries in users table reached deletion_date
	database.DeleteOldUserEntries(now)
}
