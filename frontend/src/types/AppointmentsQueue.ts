import Queue from './Queue';
import Appointment from './Appointment';
import Vue from 'vue';
import moment, { Moment } from 'moment-timezone';

// A specific slot of time that can contain any number of
// concurrent appointments.
export class AppointmentsTimeslot {
	public readonly timeslot!: number;
	public readonly time!: Moment;
	public readonly duration!: number;
	public readonly total!: number;
	public appointments: Appointment[] = [];

	constructor(
		day: number,
		startOfDay: Moment,
		timeslot: number,
		duration: number,
		total: number
	) {
		this.timeslot = timeslot;
		this.duration = duration;
		this.total = total;
		this.time = startOfDay.clone().add(timeslot * duration, 'minutes');
	}

	past(time: Moment) {
		return this.time.clone().diff(time) < 0;
	}

	get studentSlots() {
		return this.appointments.filter(
			(a) =>
				a.studentEmail !== undefined ||
				(a.staffEmail === undefined && a.studentEmail === undefined)
		);
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

	get consecutiveTimeslots(): AppointmentsTimeslot[][] {
		const groups: AppointmentsTimeslot[][] = [];
		const slots = Object.keys(this.timeslots).map((n) => parseInt(n));

		if (slots.length === 0) {
			return [];
		}

		let lastSeen: number = slots[0];
		let running: AppointmentsTimeslot[] = [this.timeslots[lastSeen]];
		for (let i = 1; i < slots.length; i++) {
			if (slots[i] !== lastSeen + 1) {
				groups.push(running);
				running = [];
			}

			running.push(this.timeslots[slots[i]]);
			lastSeen = slots[i];
		}

		groups.push(running);
		return groups;
	}

	addAppointment(appointment: Appointment) {
		// First look for appointment to update, if we already know about it
		const updated = this.updateAppointment(appointment);

		if (updated === undefined) {
			this.timeslots[appointment.timeslot].appointments.push(appointment);
		}
	}

	removeAppointment(id: string) {
		for (const slot of Object.values(this.timeslots)) {
			slot.appointments = slot.appointments.filter((a) => a.id !== id);
		}
	}

	updateAppointment(appointment: Appointment) {
		let updated = undefined;

		this.timeslots[appointment.timeslot].appointments.forEach((a, i) => {
			if (a.id === appointment.id) {
				this.timeslots[appointment.timeslot].appointments.splice(
					i,
					1,
					appointment
				);
				updated = appointment;
				return;
			}
		});

		return updated;
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
