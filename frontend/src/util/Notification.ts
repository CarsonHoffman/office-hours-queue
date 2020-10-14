function checkNotificationPromise() {
	try {
		Notification.requestPermission().then();
	} catch (e) {
		return false;
	}

	return true;
}

export default function SendNotification(title: string, body: string) {
	if ('Notification' in window) {
		if (checkNotificationPromise()) {
			Notification.requestPermission().then((p) => {
				if (p === 'granted') {
					new Notification(title, { body: body });
				}
			});
		} else {
			Notification.requestPermission((p) => {
				if (p === 'granted') {
					new Notification(title, { body: body });
				}
			});
		}
	}
}
