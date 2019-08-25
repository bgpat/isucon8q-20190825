package main

import (
	"strconv"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
)

func getUser(c echo.Context) error {
	var user User
	if err := db.QueryRow("SELECT id, nickname FROM users WHERE id = ?", c.Param("id")).Scan(&user.ID, &user.Nickname); err != nil {
		return err
	}

	loginUser, err := getLoginUser(c)
	if err != nil {
		return err
	}
	if user.ID != loginUser.ID {
		return resError(c, "forbidden", 403)
	}

	recentReservations, err := getRecentEvents(user)
	if err != nil {
		return errors.WithStack(err)
	}

	var totalPrice int
	if err := db.QueryRow("SELECT IFNULL(SUM(e.price + s.price), 0) FROM reservations r INNER JOIN sheets s ON s.id = r.sheet_id INNER JOIN events e ON e.id = r.event_id WHERE r.user_id = ? AND r.canceled_at IS NULL", user.ID).Scan(&totalPrice); err != nil {
		return err
	}

	recentEvents, err := getRecentSheets(user)
	if err != nil {
		return errors.WithStack(err)
	}

	return c.JSON(200, echo.Map{
		"id":                  user.ID,
		"nickname":            user.Nickname,
		"recent_reservations": recentReservations,
		"total_price":         totalPrice,
		"recent_events":       recentEvents,
	})
}

func getRecentEvents(user User) ([]Reservation, error) {
	recentReservations := make([]Reservation, 0)
	rows, err := db.Query("SELECT r.*, s.rank AS sheet_rank, s.num AS sheet_num FROM reservations r INNER JOIN sheets s ON s.id = r.sheet_id WHERE r.user_id = ? ORDER BY IFNULL(r.canceled_at, r.reserved_at) DESC LIMIT 5", user.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var reservation Reservation
		var sheet Sheet
		if err := rows.Scan(&reservation.ID, &reservation.EventID, &reservation.SheetID, &reservation.UserID, &reservation.ReservedAt, &reservation.CanceledAt, &sheet.Rank, &sheet.Num); err != nil {
			return nil, errors.WithStack(err)
		}

		event, err := getEvent(reservation.EventID, -1)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		price := event.Sheets[sheet.Rank].Price
		event.Sheets = nil
		event.Total = 0
		event.Remains = 0

		reservation.Event = event
		reservation.SheetRank = sheet.Rank
		reservation.SheetNum = sheet.Num
		reservation.Price = price
		reservation.ReservedAtUnix = reservation.ReservedAt.Unix()
		if reservation.CanceledAt != nil {
			reservation.CanceledAtUnix = reservation.CanceledAt.Unix()
		}
		recentReservations = append(recentReservations, reservation)
	}
	if recentReservations == nil {
		recentReservations = make([]Reservation, 0)
	}
	return recentReservations, nil
}

func getRecentSheets(user User) ([]*Event, error) {
	userIDString := "user_eventids" + strconv.FormatInt(user.ID, 10)
	eventIDs, err := client.ZRevRange(userIDString, 0, 5).Result()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var recentEvents []*Event
	for _, eventIDString := range eventIDs {
		eventID, err := strconv.ParseInt(eventIDString, 10, 64)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		event, err := getEvent(eventID, -1)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for k := range event.Sheets {
			event.Sheets[k].Detail = nil
		}
		recentEvents = append(recentEvents, event)
	}
	if recentEvents == nil {
		recentEvents = make([]*Event, 0)
	}
	return recentEvents, nil
}
