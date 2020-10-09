import Queue from './Queue';
import {QueueEntry, RemovedQueueEntry} from './QueueEntry';
import {DialogProgrammatic as Dialog, ToastProgrammatic as Toast} from 'buefy';
import moment, {Moment} from 'moment-timezone';
import linkifyStr from 'linkifyjs/string';
import g from '../main';

export default class OrderedQueue extends Queue {
	public entries: QueueEntry[] = [];
	public stack: RemovedQueueEntry[] = [];
	public schedule?: string;

	public personallyRemovedEntries = new Set<string>();

	public async pullQueueInfo() {
		return super.pullQueueInfo().then((data) => {
			this.entries = data['queue'].map((e: any) => new QueueEntry(e));
			this.stack = (data['stack'] || []).map((e: any) => new RemovedQueueEntry(e));
			this.schedule = data['schedule'];
		})
	}

	public handleWSMessage(type: string, data: any, ws: WebSocket) {
		super.handleWSMessage(type, data, ws);

		switch (type) {
			case 'ENTRY_CREATE': {
				this.addEntry(new QueueEntry(data));
				break;
			}
			case 'ENTRY_REMOVE': {
				const originalEntry = this.entries.find((e) => e.id === data.id)
				if (data.removed_by !== undefined && data.removed_by === g.$data.userInfo.email) {
					Dialog.alert({
						title: 'Popped!',
						message: `You popped ${data.email}! Their link is: ${linkifyStr(data.location)}`,
						type: 'is-success',
						hasIcon: true,
					});
				}
				else if (originalEntry !== undefined &&
					originalEntry.email === g.$data.userInfo.email &&
					!this.personallyRemovedEntries.has(data.id)) {
					Dialog.alert({
						title: 'Popped!',
						message: `You've been popped off the queue. Get ready for a staff member to join shortly!`,
						type: 'is-warning',
						hasIcon: true,
					});
				}

				this.removeEntry(data.id);
				this.addStackEntry(new RemovedQueueEntry(data));

				break;
			}
			case 'ENTRY_UPDATE': {
				const i = this.entries.findIndex((e) => e.id === data.id);
				if (i !== -1) {
					this.entries.splice(i, 1, new QueueEntry(data));
				}
				break;
			}
			case 'ENTRY_PUT_BACK': {
				this.removeStackEntry(data.id);
				break;
			}
			case 'QUEUE_CLEAR': {
				this.entries = [];
				if (data !== null) {
					Toast.open({
						duration: 60000,
						message: `${data} cleared the queue!`,
						type: 'is-danger',
					});
				} else {
					Toast.open({
						duration: 60000,
						message: 'The queue has been cleared for this session.',
						type: 'is-danger',
					});
				}
				break;
			}
		}
	}

	public addEntry(entry: QueueEntry) {
		this.entries.push(entry);
		this.entries.sort((a, b) => {
			if (a.priority != b.priority) {
				// If a's priority is higher, it should come first.
				return b.priority - a.priority;
			}

			return a.timestamp.diff(b.timestamp);
		});
	}

	public removeEntry(entryId: string) {
		this.entries = this.entries.filter(e => e.id !== entryId);
	}

	public addStackEntry(entry: RemovedQueueEntry) {
		this.stack.unshift(entry);
		this.stack.sort((a, b) => {
			return b.removedAt.diff(a.removedAt);
		});
	}

	public removeStackEntry(entryId: string) {
		this.stack = this.stack.filter(e => e.id !== entryId);
	}

	public getHalfHour(time: Moment): number {
		return Math.floor(time.clone().diff(time.clone().tz('America/New_York').startOf('day'), 'minutes') / 30);
	}

	public halfHourToTime(halfHour: number): Moment {
		return moment().tz('America/New_York').startOf('day').add(halfHour * 30, 'minutes');
	}

	public getOpenHalfHours(): number[] {
		if (this.schedule === undefined) {
			return [];
		}

		const open: number[] = [];

		for (let i = 0; i < this.schedule.length; i++) {
			if (this.schedule.charAt(i) !== 'c') {
				open.push(i);
			}
		}

		return open;
	}

	public open(time: Moment): boolean {
		return this.getOpenHalfHours().includes(this.getHalfHour(time));
	}

	public getNextOpenHalfHour(halfHour: number): number {
		const open = this.getOpenHalfHours();
		for (let i = 0; i < open.length; i++) {
			if (open[i] > halfHour) {
				return open[i];
			}
		}

		return -1;
	}

	public getNextCloseTime(halfHour: number): number {
		const open = this.getOpenHalfHours();
		const restOfDay = open.slice(open.indexOf(halfHour));

		for (let i = 1; i < restOfDay.length; i++) {
			if (restOfDay[i] - 1 > restOfDay[i - 1]) {
				return restOfDay[i - 1] + 1;
			}
		}

		return restOfDay[restOfDay.length - 1] + 1;
	}
}
