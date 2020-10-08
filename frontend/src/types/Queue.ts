import Announcement from './Announcement';
import {DialogProgrammatic as Dialog} from 'buefy';

export default class Queue {
	public readonly id!: string;
	public readonly course!: string;
	public readonly type!: 'ordered' | 'appointments';
	public readonly name!: string;
	public readonly location!: string;
	public readonly map!: string;
	public announcements: Announcement[] = [];

	constructor(data: {[index: string]: any}) {
		this.id = data['id'];
		this.course = data['course'];
		this.type = data['type'];
		this.name = data['name'];
		this.location = data['location'];
		this.map = data['map'];
	}

	public async pullQueueInfo() {
		return fetch(`/api/queues/${this.id}`).then(res => res.json()).then(data => {
			this.announcements = data['announcements'].map((a: any) => new Announcement(a));
			return data;
		});
	}

	public handleWSMessage(type: string, data: any, ws: WebSocket) {
		switch (type) {
			case 'PING': {
				ws.send(JSON.stringify({'e': 'PONG'}));
				break;
			}
			case 'MESSAGE_CREATE': {
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
