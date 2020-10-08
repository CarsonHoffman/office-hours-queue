import {DialogProgrammatic as Dialog} from 'buefy'

export default async function ErrorDialog(res: Response): Promise<any> {
	return res.json().then(data => {
		Dialog.alert({
			title: 'Request Failed',
			message: data.message,
			type: 'is-danger',
			hasIcon: true,
		});
	});
}
