import moment, { Moment } from 'moment-timezone';

export default class Appointment {
	public readonly id!: string;
	public readonly timestamp!: Moment;
	public readonly name: string | undefined;
	public readonly studentEmail: string | undefined;
	public readonly staffEmail: string | undefined;
	public readonly description: string | undefined;
	public readonly location: string | undefined;

	public readonly scheduledTime!: Moment;
	public readonly timeslot!: number;
	public readonly duration!: number;

	constructor(data: { [index: string]: any }) {
		this.id = data['id'];
		this.timestamp = moment(data['id_timestamp']);
		this.name = data['name'];
		this.studentEmail = data['student_email'];
		this.staffEmail = data['staff_email'];
		this.description = data['description'];
		this.location = data['location'];

		this.scheduledTime = moment(data['scheduled_time']);
		this.timeslot = data['timeslot'];
		this.duration = data['duration'];
	}
}
