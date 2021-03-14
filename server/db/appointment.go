package db

import (
	"context"
	"fmt"
	"time"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/segmentio/ksuid"
)

func (s *Server) GetAppointment(ctx context.Context, appointment ksuid.KSUID) (*api.AppointmentSlot, error) {
	tx := getTransaction(ctx)
	var a api.AppointmentSlot
	err := tx.GetContext(ctx, &a,
		"SELECT id, queue, staff_email, student_email, scheduled_time, timeslot, duration, name, location, description, map_x, map_y FROM appointment_slots WHERE id=$1",
		appointment,
	)
	return &a, err
}

func (s *Server) GetAppointments(ctx context.Context, queue ksuid.KSUID, from, to time.Time) ([]*api.AppointmentSlot, error) {
	tx := getTransaction(ctx)
	appointments := make([]*api.AppointmentSlot, 0)
	err := tx.SelectContext(ctx, &appointments,
		"SELECT id, queue, staff_email, student_email, scheduled_time, timeslot, duration, name, location, description, map_x, map_y FROM appointment_slots WHERE queue=$1 AND scheduled_time >= $2 AND scheduled_time <= $3 ORDER BY id",
		queue, from, to,
	)
	return appointments, err
}

func (s *Server) GetAppointmentsWithStudent(ctx context.Context, queue ksuid.KSUID, from, to time.Time) ([]*api.AppointmentSlot, error) {
	tx := getTransaction(ctx)
	appointments := make([]*api.AppointmentSlot, 0)
	err := tx.SelectContext(ctx, &appointments,
		"SELECT id, queue, timeslot, scheduled_time, duration FROM appointment_slots WHERE queue=$1 AND scheduled_time >= $2 AND scheduled_time <= $3 AND student_email IS NOT NULL ORDER BY id",
		queue, from, to,
	)
	return appointments, err
}

func (s *Server) GetAppointmentsForUser(ctx context.Context, queue ksuid.KSUID, from, to time.Time, email string) ([]*api.AppointmentSlot, error) {
	tx := getTransaction(ctx)
	appointments := make([]*api.AppointmentSlot, 0)
	err := tx.SelectContext(ctx, &appointments,
		"SELECT id, queue, student_email, scheduled_time, timeslot, duration, name, location, description, map_x, map_y FROM appointment_slots WHERE queue=$1 AND student_email=$2 AND scheduled_time >= $3 AND scheduled_time <= $4 ORDER BY id",
		queue, email, from, to,
	)
	return appointments, err
}

func (s *Server) TeammateHasAppointment(ctx context.Context, queue ksuid.KSUID, from, to time.Time, email string) (bool, error) {
	tx := getTransaction(ctx)
	var n int
	err := tx.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM appointment_slots a JOIN teammates t ON a.student_email=t.teammate WHERE t.queue=$1 AND t.email=$2 AND a.queue=$3 AND a.scheduled_time >= $4 and a.scheduled_time <= $5",
		queue, email, queue, from, to,
	)
	return n > 0, err
}

func (s *Server) GetAppointmentSchedule(ctx context.Context, queue ksuid.KSUID) ([]*api.AppointmentSchedule, error) {
	tx := getTransaction(ctx)
	schedules := make([]*api.AppointmentSchedule, 0)
	err := tx.SelectContext(ctx, &schedules, "SELECT queue, day, duration, padding, schedule FROM appointment_schedules WHERE queue=$1 ORDER BY day", queue)
	return schedules, err
}

func (s *Server) GetAppointmentScheduleForDay(ctx context.Context, queue ksuid.KSUID, day int) (*api.AppointmentSchedule, error) {
	tx := getTransaction(ctx)
	var schedule api.AppointmentSchedule
	err := tx.GetContext(ctx, &schedule, "SELECT queue, day, duration, padding, schedule FROM appointment_schedules WHERE queue=$1 AND day=$2", queue, day)
	return &schedule, err
}

func (s *Server) AddAppointmentSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule *api.AppointmentSchedule) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"INSERT INTO appointment_schedules (queue, day, duration, padding, schedule) VALUES ($1, $2, $3, $4, $5)",
		queue, day, schedule.Duration, schedule.Padding, schedule.Schedule,
	)
	return err
}

func (s *Server) UpdateAppointmentSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule *api.AppointmentSchedule) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE appointment_schedules SET duration=$1, padding=$2, schedule=$3 WHERE queue=$4 AND day=$5",
		schedule.Duration, schedule.Padding, schedule.Schedule, queue, day,
	)
	return err
}

func (s *Server) GetAppointmentsByTimeslot(ctx context.Context, queue ksuid.KSUID, from, to time.Time, timeslot int) ([]*api.AppointmentSlot, error) {
	tx := getTransaction(ctx)
	appointments := make([]*api.AppointmentSlot, 0)
	err := tx.SelectContext(ctx, &appointments,
		"SELECT id, queue, staff_email, student_email, scheduled_time, timeslot, duration, name, location, description, map_x, map_y FROM appointment_slots WHERE queue=$1 AND timeslot=$2 AND scheduled_time >= $3 AND scheduled_time <= $4 ORDER BY id",
		queue, timeslot, from, to,
	)
	return appointments, err
}

func (s *Server) ClaimTimeslot(ctx context.Context, queue ksuid.KSUID, day, timeslot int, email string) (*api.AppointmentSlot, error) {
	tx := getTransaction(ctx)
	schedule, err := s.GetAppointmentScheduleForDay(ctx, queue, day)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment schedule: %w", err)
	}

	if timeslot >= len(schedule.Schedule) {
		return nil, fmt.Errorf("attempted to claim slot %d out of %d slots", timeslot, len(schedule.Schedule))
	}

	from, to := api.WeekdayBounds(day)
	slots, err := s.GetAppointmentsByTimeslot(ctx, queue, from, to, timeslot)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment slots: %w", err)
	}

	// Check if there's an existing slot without a staff member; if so,
	// prefer taking that one first
	for _, slot := range slots {
		if slot.StaffEmail == nil {
			var a api.AppointmentSlot
			err := tx.GetContext(ctx, &a,
				"UPDATE appointment_slots SET staff_email=$1 WHERE id=$2 RETURNING *",
				email, slot.ID,
			)
			return &a, err
		}
	}

	// If we made it here, there aren't any existing slots without a
	// staff member. Now check if there are any open spots
	open := int(schedule.Schedule[timeslot]-'0') - len(slots)
	if open < 1 {
		return nil, fmt.Errorf("no spots open to claim at timeslot %d", timeslot)
	}

	// There's room for another appointment at the current timeslot.
	// Let's claim it.
	id := ksuid.New()
	appointmentTime := api.TimeslotToTime(day, timeslot, schedule.Duration)
	var a api.AppointmentSlot
	err = tx.GetContext(ctx, &a,
		"INSERT INTO appointment_slots (id, queue, staff_email, scheduled_time, timeslot, duration) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *",
		id, queue, email, appointmentTime, timeslot, schedule.Duration,
	)
	return &a, err
}

func (s *Server) UnclaimAppointment(ctx context.Context, appointment ksuid.KSUID) (deleted bool, err error) {
	tx := getTransaction(ctx)
	a, err := s.GetAppointment(ctx, appointment)
	if err != nil {
		return false, fmt.Errorf("failed to get appointment: %w", err)
	}

	// If there's no student associated with this appointment, there's
	// no point in keeping it around
	if a.StudentEmail == nil {
		_, err = tx.ExecContext(ctx,
			"DELETE FROM appointment_slots WHERE id=$1",
			appointment,
		)
		return true, err
	}

	// If there is a student associated with it, just remove the staff email
	_, err = tx.ExecContext(ctx,
		"UPDATE appointment_slots SET staff_email=NULL WHERE id=$1",
		appointment,
	)
	return false, err
}

func (s *Server) SignupForAppointment(ctx context.Context, queue ksuid.KSUID, appointment *api.AppointmentSlot) (*api.AppointmentSlot, error) {
	tx := getTransaction(ctx)
	start, end := api.WeekdayBounds(int(appointment.ScheduledTime.Local().Weekday()))
	var newAppointment api.AppointmentSlot
	appointments, err := s.GetAppointmentsByTimeslot(ctx, queue, start, end, appointment.Timeslot)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments for timeslot: %w", err)
	}

	// Check if an appointment without a student already exists
	for _, a := range appointments {
		if a.StudentEmail == nil {
			err = tx.GetContext(ctx, &newAppointment,
				"UPDATE appointment_slots SET student_email=$1, name=$2, location=$3, description=$4, map_x=$5, map_y=$6 WHERE id=$7 RETURNING id, queue, student_email, staff_email, scheduled_time, timeslot, duration, name, location, description, map_x, map_y",
				*appointment.StudentEmail, *appointment.Name, *appointment.Location, *appointment.Description, *appointment.MapX, *appointment.MapY, a.ID,
			)
			return &newAppointment, err
		}
	}

	// If not, insert a new appointment
	id := ksuid.New()
	err = tx.GetContext(ctx, &newAppointment,
		"INSERT INTO appointment_slots (id, queue, student_email, scheduled_time, timeslot, duration, name, location, description, map_x, map_y) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id, queue, student_email, scheduled_time, timeslot, duration, name, location, description, map_x, map_y",
		id, appointment.Queue, appointment.StudentEmail, appointment.ScheduledTime, appointment.Timeslot, appointment.Duration, appointment.Name, appointment.Location, appointment.Description, appointment.MapX, appointment.MapY,
	)
	return &newAppointment, err
}

func (s *Server) UpdateAppointment(ctx context.Context, appointment ksuid.KSUID, newAppointment *api.AppointmentSlot) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE appointment_slots SET name=$1, location=$2, description=$3, map_x=$4, map_y=$5 WHERE id=$6",
		newAppointment.Name, newAppointment.Location, newAppointment.Description, newAppointment.MapX, newAppointment.MapY, appointment,
	)
	return err
}

func (s *Server) RemoveAppointmentSignup(ctx context.Context, appointment ksuid.KSUID) (deleted bool, newAppointment *api.AppointmentSlot, err error) {
	tx := getTransaction(ctx)
	a, err := s.GetAppointment(ctx, appointment)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get appointment: %w", err)
	}

	// If there's no staff member associated with this appointment, just drop it
	if a.StaffEmail == nil {
		_, err = tx.ExecContext(ctx,
			"DELETE FROM appointment_slots WHERE id=$1",
			appointment,
		)
		return true, nil, err
	}

	// If a staff member has a claim on this appointment, don't delete it,
	// just set the student fields to null
	var newAppt api.AppointmentSlot
	err = tx.GetContext(ctx, &newAppt,
		"UPDATE appointment_slots SET student_email=NULL, name=NULL, location=NULL, description=NULL, map_x=NULL, map_y=NULL WHERE id=$1 RETURNING *",
		appointment,
	)
	return false, &newAppt, err
}
