package main

import (
	"os"
	"os/exec"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func initialize(c echo.Context) error {
	if err := initializeDB(); err != nil {
		return errors.WithStack(err)
	}

	client.FlushAll()

	userEvents := make(map[int64][]redis.Z)
	events := make(map[int64]map[string]interface{})

	rowsReservations, err := db.Query("SELECT * FROM reservations")
	if err != nil {
		return nil
	}

	for rowsReservations.Next() {
		var reservation Reservation
		if err = rowsReservations.Scan(&reservation.ID, &reservation.EventID, &reservation.SheetID, &reservation.UserID, &reservation.ReservedAt, &reservation.CanceledAt); err != nil {
			return nil
		}
		sheetIDString := strconv.FormatInt(reservation.SheetID, 10)
		jsonData, err := (&ReservationsRedisType{
			UserID:     reservation.UserID,
			ReservedAt: *reservation.ReservedAt,
		}).Marshal()
		if err != nil {
			return err
		}

		if _, ok := events[reservation.EventID]; !ok {
			events[reservation.EventID] = make(map[string]interface{})
		}
		events[reservation.EventID][sheetIDString] = string(jsonData)

		userEvents[reservation.UserID] = append(userEvents[reservation.UserID], redis.Z{
			Score:  (float64)(reservation.ReservedAt.UnixNano()),
			Member: reservation.EventID,
		})
	}

	var eg errgroup.Group

	for id, events := range events {
		eventIDString := "event" + strconv.FormatInt(id, 10)
		events := events
		eg.Go(func() error {
			return client.HMSet(eventIDString, events).Err()
		})
	}

	for id, z := range userEvents {
		userIDString := "user_eventids" + strconv.FormatInt(id, 10)
		z := z
		eg.Go(func() error {
			return client.ZAdd(userIDString, z...).Err()
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.WithStack(err)
	}

	return c.NoContent(204)
}

func initializeDB() error {
	cmd := exec.Command("../../db/init.sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return errors.WithStack(cmd.Run())
}
