import Queue from './Queue';
import OrderedQueue from './OrderedQueue';
import { AppointmentsQueue } from './AppointmentsQueue';

export default class Course {
	public readonly id: string;
	public readonly shortName: string;
	public readonly fullName: string;

	public readonly queues: Queue[] = [];

	public favorite = false;

	constructor(data: { [index: string]: any }) {
		this.id = data['id'];
		this.shortName = data['short_name'];
		this.fullName = data['full_name'];
		this.queues = data['queues'].map((q: any) => {
			switch (q.type) {
				case 'ordered': {
					return new OrderedQueue(q, this);
				}
				case 'appointments': {
					return new AppointmentsQueue(q, this);
				}
				default: {
					return undefined;
				}
			}
		});
	}
}
