export default class Appoinmentlog {
    public readonly date!: string;
    public readonly available!: string;
    public readonly used!: string;

    constructor(data: { [index: string]: any }) {
        this.date = data['date'];
        this.available = data['available'];
        this.used = data['used'];
    }
}
