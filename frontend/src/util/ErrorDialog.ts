import { DialogProgrammatic as Dialog } from 'buefy';
import EscapeHTML from '@/util/Sanitization';

export default async function ErrorDialog(res: Response): Promise<any> {
	return res.json().then((data) => {
		Dialog.alert({
			title: 'Request Failed',
			message: EscapeHTML(data.message),
			type: 'is-danger',
			hasIcon: true,
		});
	});
}
