import moment, {Moment} from 'moment';

export class QueueEntry {
	public readonly id!: string;
	public readonly timestamp!: Moment;
	public readonly name!: string;
	public readonly email!: string;
	public readonly description!: string;
	public readonly location!: string;
	public readonly priority: number = 0;
	public readonly pinned: boolean = false;

	constructor(data: {[index: string]: any}) {
		this.id = data['id'];
		this.timestamp = moment(data['id_timestamp']);
		this.name = data['name'];
		this.email = data['email'];
		this.description = data['description'];
		this.location = data['location'];
		this.priority = data['priority'];
		this.pinned = data['pinned'] !== undefined || false;
	}

	// Get the humanized timestamp in relation to time.
	// We pass in a parameter here instead of using moment()
	// to overcome reactivity issues between Vue and moment.
	public humanizedTimestamp(time: Moment): string {
		return this.timestamp.from(time);
	}

	get tooltipTimestamp(): string {
		return this.timestamp.format(
			'YYYY-MM-DD h:mm:ss a'
		);
	}
}

export class RemovedQueueEntry extends QueueEntry {
	public readonly removedAt!: Moment;
	public readonly removedBy!: string;

	constructor(data: {[index: string]: any}) {
		super(data);
		this.removedAt = moment(data['removed_at']);
		this.removedBy = data['removed_by'];
	}

	humanizedTimestamp(time: Moment): string {
		return this.removedAt.from(time);
	}

	get tooltipTimestamp() {
		return this.removedAt.format(
			'YYYY-MM-DD h:mm:ss a'
		);
	}
}
