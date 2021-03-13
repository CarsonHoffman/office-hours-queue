import Queue from './Queue';
import { Appointment, AppointmentSlot } from './Appointment';
import Vue from 'vue';
import moment, { Moment } from 'moment-timezone';

// A specific slot of time that can contain any number of
// concurrent appointments.
export class AppointmentsTimeslot {
	public readonly timeslot!: number;
	public readonly time!: Moment;
	public readonly duration!: number;
	public slots: AppointmentSlot[] = [];

	constructor(
		day: number,
		startOfDay: Moment,
		timeslot: number,
		duration: number,
		total: number
	) {
		this.timeslot = timeslot;
		this.duration = duration;
		// We need to calculate the hour manually instead of just using minutes
		// for daylight savings purposes (if the appointment was usually at 10 AM,
		// we do not want it to occur at 9 AM or 11 AM)
		this.time = startOfDay
			.clone()
			.hour(Math.floor((timeslot * duration) / 60))
			.minute((timeslot * duration) % 60);
		this.slots = new Array(total);
		this.slots.fill(new AppointmentSlot(this.time, timeslot, duration));
	}

	past(time: Moment) {
		return this.time.clone().diff(time) < 0;
	}

	addAppointment(appointment: Appointment) {
		const toAddIndex = this.slots.findIndex((s) => !s.filled);
		if (toAddIndex !== -1) {
			this.slots.splice(toAddIndex, 1, appointment);
		} else {
			this.slots.push(appointment);
		}
	}

	removeAppointment(id: string) {
		const toRemoveIndex = this.slots.findIndex(
			(s) => s.filled && (s as Appointment).id === id
		);
		if (toRemoveIndex !== -1) {
			this.slots.splice(
				toRemoveIndex,
				1,
				new AppointmentSlot(this.time, this.timeslot, this.duration)
			);
			return id;
		}
		return undefined;
	}

	updateAppointment(appointment: Appointment) {
		const i = this.slots.findIndex(
			(s) => s.filled && (s as Appointment).id === appointment.id
		);
		if (i !== -1) {
			this.slots.splice(i, 1, appointment);
			return appointment;
		}
		return undefined;
	}
}

export class AppointmentsSchedule {
	public readonly day!: number;
	public readonly duration!: number;
	public readonly padding!: number;
	public readonly timeslots: { [index: number]: AppointmentsTimeslot } = {};

	constructor(
		day: number,
		startOfDay: Moment,
		duration: number,
		padding: number,
		schedule: string
	) {
		this.day = day;
		this.duration = duration;
		this.padding = padding;
		schedule.split('').forEach((v, i) => {
			const total = parseInt(v);
			if (total > 0) {
				Vue.set(
					this.timeslots,
					i,
					new AppointmentsTimeslot(day, startOfDay, i, this.duration, total)
				);
			}
		});
	}

	get numSlots() {
		return Object.keys(this.timeslots).length;
	}

	// Returns a map of timeslots to appointments.
	// Would really like to do this more functionally but the object
	// seems to make things hard :(
	get appointmentSlots(): { [index: number]: AppointmentSlot[] } {
		const appointments: { [index: number]: AppointmentSlot[] } = {};
		for (const [index, slot] of Object.entries(this.timeslots)) {
			const slotIndex = parseInt(index);
			appointments[slotIndex] = slot.slots;
		}

		return appointments;
	}

	addAppointment(appointment: Appointment) {
		// First look for appointment to update, if we already know about it
		const updated = this.timeslots[appointment.timeslot].updateAppointment(
			appointment
		);

		if (updated === undefined) {
			this.timeslots[appointment.timeslot].addAppointment(appointment);
		}
	}

	// Yes, this is slow. At least it's rare. :)
	removeAppointment(id: string) {
		Object.values(this.timeslots).forEach((slot) => {
			const removed = slot.removeAppointment(id);
			if (removed !== undefined) {
				return;
			}
		});
	}

	updateAppointment(appointment: Appointment) {
		this.timeslots[appointment.timeslot].updateAppointment(appointment);
	}
}

export class AppointmentsQueue extends Queue {
	public schedule: AppointmentsSchedule | undefined;

	public handleWSMessage(type: string, data: any, ws: WebSocket) {
		super.handleWSMessage(type, data, ws);

		switch (type) {
			case 'APPOINTMENT_CREATE': {
				if (this.schedule !== undefined) {
					this.schedule.addAppointment(new Appointment(data));
				}
				break;
			}
			case 'APPOINTMENT_REMOVE': {
				if (this.schedule !== undefined) {
					this.schedule.removeAppointment(data['id']);
				}
				break;
			}
			case 'APPOINTMENT_UPDATE': {
				if (this.schedule !== undefined) {
					this.schedule.updateAppointment(new Appointment(data));
				}
			}
		}
	}

	pullQueueInfo(time: Moment) {
		return Promise.all([
			super.pullQueueInfo(time),
			fetch(
				process.env.BASE_URL +
					`api/queues/${this.id}/appointments/schedule/${this.day(time)}`
			),
			fetch(
				process.env.BASE_URL +
					`api/queues/${this.id}/appointments/${this.day(time)}`
			),
		])
			.then(([_, schedule, appointments]) =>
				Promise.all([schedule.json(), appointments.json()])
			)
			.then(([schedule, appointments]) => {
				Vue.set(
					this,
					'schedule',
					new AppointmentsSchedule(
						this.day(time),
						time
							.clone()
							.tz('America/New_York')
							.startOf('day'),
						schedule['duration'],
						schedule['padding'],
						schedule['schedule']
					)
				);

				appointments.forEach((v: any) => {
					this.schedule?.addAppointment(new Appointment(v));
				});
			});
	}

	day(time: Moment) {
		return time
			.clone()
			.tz('America/New_York')
			.day();
	}
}
