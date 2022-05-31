import moment, { Moment } from 'moment-timezone';

export class AppointmentSlot {
	public readonly scheduledTime!: Moment;
	public readonly timeslot!: number;
	public readonly duration!: number;

	constructor(time: Moment, timeslot: number, duration: number) {
		this.scheduledTime = time;
		this.timeslot = timeslot;
		this.duration = duration;
	}

	get filled() {
		return false;
	}

	get filledByStudent() {
		return false;
	}

	get filledByStaff() {
		return false;
	}

	past(time: Moment) {
		return this.scheduledTime < time;
	}
}

export class Appointment extends AppointmentSlot {
	public readonly id!: string;
	public readonly timestamp!: Moment;
	public readonly name: string | undefined;
	public readonly studentEmail: string | undefined;
	public readonly staffEmail: string | undefined;
	public readonly description: string | undefined;
	public readonly location: string | undefined;

	constructor(data: { [index: string]: any }) {
		super(
			moment(data['scheduled_time']).local(),
			data['timeslot'],
			data['duration']
		);

		this.id = data['id'];
		this.timestamp = moment(data['id_timestamp']).local();
		this.name = data['name'];
		this.studentEmail = data['student_email'];
		this.staffEmail = data['staff_email'];
		this.description = data['description'];
		this.location = data['location'];
	}

	get filled() {
		return true;
	}

	get filledByStudent() {
		// If this appointment exists and both the staff and student emails are
		// undefined, it must be filled by a student (and we don't have permission
		// to know who).
		return this.studentEmail !== undefined || this.staffEmail === undefined;
	}

	get filledByStaff() {
		// This isn't relevant for non-admins.
		return this.staffEmail !== undefined;
	}
}
