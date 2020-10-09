export default function SendNotification(title: string, body: string) {
	Notification.requestPermission().then((p) => {
		if (p === 'granted') {
			new Notification(title, {body: body});
		}
	})
}
