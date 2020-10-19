import Queue from './Queue';
import { QueueEntry, RemovedQueueEntry } from './QueueEntry';
import SendNotification from '../util/Notification';
import {
	DialogProgrammatic as Dialog,
	ToastProgrammatic as Toast,
} from 'buefy';
import moment, { Moment } from 'moment-timezone';
import linkifyStr from 'linkifyjs/string';
import g from '../main';

export default class OrderedQueue extends Queue {
	public entries: QueueEntry[] = [];
	public stack: RemovedQueueEntry[] = [];
	public schedule?: string;

	public personallyRemovedEntries = new Set<string>();

	public async pullQueueInfo(time: Moment) {
		return super.pullQueueInfo(time).then((data) => {
			this.entries = data['queue'].map((e: any) => new QueueEntry(e));
			this.stack = (data['stack'] || []).map(
				(e: any) => new RemovedQueueEntry(e)
			);
			this.schedule = data['schedule'];
			this.setDocumentTitle();
		});
	}

	public setDocumentTitle() {
		document.title = `${this.course.shortName} Office Hours (${this.entries.length})`;
	}

	public handleWSMessage(type: string, data: any, ws: WebSocket) {
		super.handleWSMessage(type, data, ws);

		switch (type) {
			case 'ENTRY_CREATE': {
				const existing = this.entries.findIndex((e) => e.id === data.id);
				if (existing !== -1) {
					this.entries.splice(existing, 1, new QueueEntry(data));
					this.sortEntries();
					return;
				}

				if (
					g.$data.userInfo.admin_courses !== undefined &&
					g.$data.userInfo.admin_courses.includes(this.course.id)
				) {
					Toast.open({
						duration: 2000,
						message: `${data.email} joined the queue!`,
						type: 'is-primary',
					});

					if (this.entries.length === 0) {
						SendNotification(
							'A new student joined the queue!',
							`A wild ${data.email} has appeared!`
						);
					}
				}

				this.addEntry(new QueueEntry(data));
				break;
			}
			case 'ENTRY_REMOVE': {
				const originalEntry = this.entries.find((e) => e.id === data.id);
				if (
					data.removed_by !== undefined &&
					data.removed_by === g.$data.userInfo.email
				) {
					Dialog.alert({
						title: 'Popped!',
						message: `You popped ${data.email}! Their link is: ${linkifyStr(
							data.location
						)}`,
						type: 'is-success',
						hasIcon: true,
					});
				} else if (
					originalEntry !== undefined &&
					originalEntry.email !== undefined &&
					originalEntry.email === g.$data.userInfo.email &&
					!this.personallyRemovedEntries.has(data.id)
				) {
					SendNotification(
						'You were popped!',
						`Please be ready for a staff member to join your meeting!`
					);
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
				this.sortEntries();
				break;
			}
			case 'ENTRY_PINNED': {
				SendNotification(
					'You were pinned!',
					'Another staff member will be joining shortly!'
				);
				Dialog.alert({
					title: 'Pinned!',
					message:
						`You were pinned on the queue! More help is on the way. ` +
						`You'll get a notification when you've been popped again.`,
					type: 'is-info',
					hasIcon: true,
				});
				break;
			}
			case 'STACK_REMOVE': {
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

		this.setDocumentTitle();
	}

	public addEntry(entry: QueueEntry) {
		this.entries.push(entry);
		this.sortEntries();
	}

	public sortEntries() {
		this.entries.sort((a, b) => {
			if (a.pinned != b.pinned) {
				return a.pinned ? -1 : 1;
			}

			if (a.priority != b.priority) {
				// If a's priority is higher, it should come first.
				return b.priority - a.priority;
			}

			return a.timestamp.diff(b.timestamp);
		});
	}

	public removeEntry(entryId: string) {
		this.entries = this.entries.filter((e) => e.id !== entryId);
	}

	public addStackEntry(entry: RemovedQueueEntry) {
		this.stack.unshift(entry);
		this.stack.sort((a, b) => {
			return b.removedAt.diff(a.removedAt);
		});
	}

	public removeStackEntry(entryId: string) {
		this.stack = this.stack.filter((e) => e.id !== entryId);
	}

	public getHalfHour(time: Moment): number {
		return Math.floor(
			time.clone().diff(
				time
					.clone()
					.tz('America/New_York')
					.startOf('day'),
				'minutes'
			) / 30
		);
	}

	public halfHourToTime(halfHour: number): Moment {
		return moment()
			.tz('America/New_York')
			.startOf('day')
			.add(halfHour * 30, 'minutes');
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
