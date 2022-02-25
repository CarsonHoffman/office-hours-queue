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
import ErrorDialog from '@/util/ErrorDialog';

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
			this.online.forEach((email: string) => {
				this.entries
					.filter((e: QueueEntry) => e.email === email)
					.forEach((e: QueueEntry) => {
						e.online = true;
					});
				this.stack
					.filter((e: QueueEntry) => e.email === email)
					.forEach((e: QueueEntry) => {
						e.online = true;
					});
			});
			this.setDocumentTitle();
		});
	}

	public setDocumentTitle() {
		let title = '';
		if (this.entries.length > 0) {
			title += '(';
			const pos = this.entryIndex(g.$data.userInfo.email);
			if (pos !== -1) {
				title += `#${pos + 1} of `;
			}
			title += `${this.entries.length}) `;
		}
		title += `${this.course.shortName} Office Hours`;
		document.title = title;
	}

	get admin(): boolean {
		return (
			!g.studentView &&
			g.$data.userInfo.admin_courses !== undefined &&
			g.$data.userInfo.admin_courses.includes(this.course.id)
		);
	}

	public handleWSMessage(type: string, data: any, ws: WebSocket) {
		super.handleWSMessage(type, data, ws);

		switch (type) {
			case 'ENTRY_CREATE': {
				if (data.email !== undefined) {
					data.online = this.online.has(data.email);
				}
				const existing = this.entries.findIndex((e) => e.id === data.id);
				if (existing !== -1) {
					this.entries.splice(existing, 1, new QueueEntry(data));
					this.sortEntries();
					return;
				}

				if (this.admin) {
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
					this.admin &&
					data.removed_by !== undefined &&
					data.removed_by === g.$data.userInfo.email
				) {
					Dialog.confirm({
						title: 'Popped!',
						message:
							`You popped ${data.email}! Their link is: ${linkifyStr(
								data.location
							)}.` +
							`<br><br>If you weren't able to make contact with the student, click "Not Helped" to alert the student that you attempted to help them.` +
							(this.config?.prioritizeNew
								? ` Additionally, this won't count against their first-help-of-the-day status.`
								: ''),
						type: 'is-success',
						hasIcon: true,
						canCancel: ['button'],
						cancelText: 'Not Helped',
						onCancel: () => {
							fetch(
								process.env.BASE_URL +
									`api/queues/${this.id}/entries/${data.id}/helped`,
								{
									method: 'DELETE',
								}
							).then((res) => {
								if (res.status !== 204) {
									return ErrorDialog(res);
								}

								Toast.open({
									duration: 5000,
									message: 'Successfully set that student was not helped!',
									type: 'is-success',
								});
							});
						},
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
				if (data.email !== undefined) {
					data.online = this.online.has(data.email);
				}
				this.addStackEntry(new RemovedQueueEntry(data));

				break;
			}
			case 'ENTRY_UPDATE': {
				if (data.email !== undefined) {
					data.online = this.online.has(data.email);
				}
				const i = this.entries.findIndex((e) => e.id === data.id);
				if (i !== -1) {
					this.entries.splice(i, 1, new QueueEntry(data));
					this.sortEntries();
				} else {
					console.log('hi stack');
					const stackIndex = this.stack.findIndex((e) => e.id === data.id);
					if (stackIndex !== -1) {
						console.log('inner');
						this.stack.splice(stackIndex, 1, new RemovedQueueEntry(data));
					}
				}
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
				// Estimate what the stack will look like based on
				// the information received from the event. The removed
				// time might differ by a second or so versus when the
				// user refreshes, but this should work fine.
				const removed = this.entries.map((e) =>
					RemovedQueueEntry.fromEntry(e, moment(), data)
				);
				this.entries = [];
				this.stack.unshift(...removed);
				this.sortStack();
				if (this.admin && data !== null) {
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
			case 'NOT_HELPED': {
				Dialog.alert({
					title: `We Couldn't Find You!`,
					message:
						`A staff member attempted to help you, but they let us know that they weren't able to make contact with you. Please make sure your location is descriptive or your meeting link is still valid!` +
						(this.config?.prioritizeNew
							? `<br><br><b>This didn't count as your first meeting of the day.</b>`
							: ''),
					hasIcon: true,
					type: 'is-danger',
				});
				break;
			}
			case 'USER_STATUS_UPDATE': {
				const email = data.email;
				const online = data.status === 'online';
				if (online) {
					this.online.add(email);
				} else {
					this.online.delete(email);
				}
				this.entries
					.filter((e: QueueEntry) => e.email === email)
					.forEach((e: QueueEntry) => {
						e.online = online;
					});
				this.stack
					.filter((e: QueueEntry) => e.email === email)
					.forEach((e: QueueEntry) => {
						e.online = online;
					});
				break;
			}
		}

		this.setDocumentTitle();
	}

	public addEntry(entry: QueueEntry) {
		console.log('pushing');
		this.entries.push(entry);
		console.log('pushed');
		this.sortEntries();
		console.log('sorted');
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

			return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
		});
	}

	public sortStack() {
		this.stack.sort((a, b) => {
			if (a.removedAt != b.removedAt) {
				return b.removedAt.clone().diff(a.removedAt);
			}
			return a.id > b.id ? -1 : a.id < b.id ? 1 : 0;
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
		// This represents the half hour with regard to the normal
		// 48-half-hour schedule, not necessarily the index in the day
		// (looking at you, daylight savings)
		return Math.floor(
			(time
				.clone()
				.tz('America/New_York')
				.hour() *
				60 +
				time
					.clone()
					.tz('America/New_York')
					.minute()) /
				30
		);
	}

	public halfHourToTime(halfHour: number): Moment {
		// We need to calculate the hour manually instead of just using minutes
		// for daylight savings purposes (if the half hour was usually at 10 AM,
		// we do not want it to occur at 9 AM or 11 AM)
		return moment()
			.tz('America/New_York')
			.startOf('day')
			.hour(Math.floor(halfHour / 2))
			.minute((halfHour % 2) * 30);
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

	public entryIndex(email: string | undefined): number {
		if (email === undefined) {
			return -1;
		}

		return this.entries.findIndex((e) => e.email === email);
	}

	public entry(email: string | undefined): QueueEntry | null {
		const i = this.entryIndex(email);
		return i !== -1 ? this.entries[i] : null;
	}
}
