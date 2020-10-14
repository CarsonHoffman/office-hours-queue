import moment, { Moment } from 'moment';

export class QueueEntry {
	public readonly id!: string;
	public readonly timestamp!: Moment;
	public readonly name: string | undefined;
	public readonly email: string | undefined;
	public readonly description: string | undefined;
	public readonly location: string | undefined;
	public readonly priority!: number;
	public readonly pinned!: boolean;

	constructor(data: { [index: string]: any }) {
		this.id = data['id'];
		this.timestamp = moment(data['id_timestamp']);
		this.name = data['name'];
		this.email = data['email'];
		this.description = data['description'];
		this.location = data['location'];
		this.priority = data['priority'] || 0;
		this.pinned = data['pinned'] || false;
	}

	// Get the humanized timestamp in relation to time.
	// We pass in a parameter here instead of using moment()
	// to overcome reactivity issues between Vue and moment.
	public humanizedTimestamp(time: Moment): string {
		return this.timestamp.from(time);
	}

	get tooltipTimestamp(): string {
		return this.timestamp.format('YYYY-MM-DD h:mm:ss a');
	}
}

export class RemovedQueueEntry extends QueueEntry {
	public readonly removedAt!: Moment;
	public readonly removedBy!: string;

	constructor(data: { [index: string]: any }) {
		super(data);
		this.removedAt = moment(data['removed_at']);
		this.removedBy = data['removed_by'];
	}

	humanizedTimestamp(time: Moment): string {
		return this.removedAt.from(time);
	}

	get tooltipTimestamp() {
		return this.removedAt.format('YYYY-MM-DD h:mm:ss a');
	}
}
