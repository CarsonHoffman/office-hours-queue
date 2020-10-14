export default class Announcement {
	public readonly id!: string;
	public readonly queue!: string;
	public readonly content!: string;

	constructor(data: { [index: string]: any }) {
		this.id = data['id'];
		this.queue = data['queue'];
		this.content = data['content'];
	}
}
