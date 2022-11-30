import Announcement from './Announcement';
import Course from './Course';
import SendNotification from '../util/Notification';
import EscapeHTML from '../util/Sanitization';
import { DialogProgrammatic as Dialog } from 'buefy';
import moment, { Moment } from 'moment-timezone';

export class QueueConfiguration {
	public virtual: boolean | undefined;
	public confirmSignupMessage: string | undefined;
	public enableLocationField: boolean | undefined;
	public preventGroups: boolean | undefined;
	public preventGroupsBoost: boolean | undefined;
	public preventUnregistered: boolean | undefined;
	public prioritizeNew: boolean | undefined;
	public scheduled: boolean | undefined;

	constructor(data: { [index: string]: any }) {
		this.virtual = data['virtual'];
		this.confirmSignupMessage = data['confirm_signup_message'];
		this.enableLocationField = data['enable_location_field'];
		this.preventGroups = data['prevent_groups'];
		this.preventGroupsBoost = data['prevent_groups_boost'];
		this.preventUnregistered = data['prevent_unregistered'];
		this.prioritizeNew = data['prioritize_new'];
		this.scheduled = data['scheduled'];
	}
}

export default class Queue {
	public readonly id!: string;
	public readonly type!: 'ordered' | 'appointments';
	public readonly name!: string;
	public readonly location!: string;
	public readonly map!: string;
	public announcements: Announcement[] = [];

	public config: QueueConfiguration | null;

	public course!: Course;

	public websocketConnections = 0;
	public online: Set<string>;

	constructor(data: { [index: string]: any }, course: Course) {
		this.id = data['id'];
		this.type = data['type'];
		this.name = data['name'];
		this.location = data['location'];
		this.map = data['map'];

		this.course = course;
		this.online = new Set<string>();
		this.config = null;
	}

	public async pullQueueInfo(time: Moment) {
		return fetch(process.env.BASE_URL + `api/queues/${this.id}`)
			.then((res) => res.json())
			.then((data) => {
				this.announcements = data['announcements'].map(
					(a: any) => new Announcement(a)
				);
				this.config = new QueueConfiguration(data['config']);
				if (data.online !== undefined) {
					this.online = new Set(data.online);
				}

				return data;
			});
	}

	public handleWSMessage(type: string, data: any, ws: WebSocket) {
		switch (type) {
			case 'PING': {
				ws.send(JSON.stringify({ e: 'PONG' }));
				break;
			}
			case 'MESSAGE_CREATE': {
				const broadcast = data.receiver === '<broadcast>';
				const title = `Message from ${this.course.shortName} Staff`;
				SendNotification(title, data.content);
				Dialog.alert({
					title: title,
					message: EscapeHTML(data.content),
					type: 'is-warning',
					hasIcon: true,
					icon: broadcast ? 'bullhorn' : 'envelope-open-text',
				});

				break;
			}
			case 'ANNOUNCEMENT_CREATE': {
				this.announcements.push(new Announcement(data));
				break;
			}
			case 'ANNOUNCEMENT_DELETE': {
				this.announcements = this.announcements.filter((a) => a.id !== data);
				break;
			}
			case 'REFRESH': {
				// Pick random delay to help mitigate thundering herd on server
				const delay = Math.random() * 30000;
				Dialog.alert({
					title: 'Refreshing Shortly',
					message: `The server told me that we need to refresh the page to get new information. Refreshing in ${EscapeHTML(
						moment.duration(delay).humanize()
					)}…`,
					type: 'is-warning',
					hasIcon: true,
				});
				setTimeout(() => location.reload(), delay);
				break;
			}
			case 'QUEUE_RANDOMIZE': {
				Dialog.alert({
					title: 'Queue Randomized',
					message:
						'The order of the queue was just randomized. The priorities on the queue now correspond to that randomization.',
					type: 'is-warning',
					hasIcon: true,
				});
				break;
			}
			case 'QUEUE_CONNECTIONS_UPDATE': {
				this.websocketConnections = data;
				break;
			}
		}
	}
}
