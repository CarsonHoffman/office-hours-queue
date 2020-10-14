import Announcement from './Announcement';
import Course from './Course';
import SendNotification from '../util/Notification';
import { DialogProgrammatic as Dialog } from 'buefy';

export default class Queue {
	public readonly id!: string;
	public readonly type!: 'ordered' | 'appointments';
	public readonly name!: string;
	public readonly location!: string;
	public readonly map!: string;
	public announcements: Announcement[] = [];

	public course!: Course;

	constructor(data: { [index: string]: any }, course: Course) {
		this.id = data['id'];
		this.type = data['type'];
		this.name = data['name'];
		this.location = data['location'];
		this.map = data['map'];

		this.course = course;
	}

	public async pullQueueInfo() {
		return fetch(process.env.BASE_URL + `api/queues/${this.id}`)
			.then((res) => res.json())
			.then((data) => {
				this.announcements = data['announcements'].map(
					(a: any) => new Announcement(a)
				);
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
				SendNotification(
					`Message from ${this.course.shortName} Staff`,
					data.content
				);
				Dialog.alert({
					title: 'Message from Staff',
					message: data.content,
					type: 'is-warning',
					hasIcon: true,
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
		}
	}
}
