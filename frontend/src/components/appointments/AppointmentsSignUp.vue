<template>
	<div>
		<div class="field">
			<label class="label">Name</label>
			<div class="control has-icons-left">
				<input class="input" v-model="name" type="text" placeholder="Nice to meet you!" />
				<span class="icon is-small is-left">
					<font-awesome-icon icon="user" />
				</span>
			</div>
		</div>
		<div class="field">
			<label class="label">Description</label>
			<div class="control has-icons-left">
				<input
					class="input"
					v-model="description"
					type="text"
					placeholder="Help us help you—please be descriptive!"
				/>
				<span class="icon is-small is-left">
					<font-awesome-icon icon="question" />
				</span>
			</div>
		</div>
		<div class="field">
			<label class="label">Meeting Link</label>
			<div class="control has-icons-left">
				<input class="input" v-model="location" type="text" />
				<span class="icon is-small is-left">
					<font-awesome-icon icon="link" />
				</span>
			</div>
		</div>
		<div class="field">
			<div class="control level-left">
				<button
					class="button is-success level-item"
					v-if="selectedTimeslot === null"
					disabled
				>Select a time slot!</button>
				<button
					class="button is-success level-item"
					:class="{'is-loading': signUpRequstRunning}"
					:disabled="!canSignUp"
					v-else-if="myAppointment === undefined"
					@click="signUp"
				>Sign up for appointment at {{selectedTime.format('LT')}}</button>
				<button
					class="button is-warning level-item"
					:class="{'is-loading': updateAppointmentRequestRunning}"
					v-else-if="myAppointmentModified"
					@click="updateAppointment"
				>Update Appointment</button>
				<button
					class="button is-success level-item"
					disabled="true"
					v-else
				>Signed up for {{myAppointment.scheduledTime.format('LT')}}</button>
				<button
					class="button is-danger level-item"
					:class="{'is-loading': cancelAppointmentRequestRunning}"
					v-if="myAppointment !== undefined"
					@click="cancelAppointment"
				>Cancel Appointment</button>
				<p class="level-item" v-if="!$root.$data.loggedIn">Log in to sign up!</p>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { Moment } from 'moment';
import { Component, Prop, Watch } from 'vue-property-decorator';
import { AppointmentsQueue } from '@/types/AppointmentsQueue';
import Appointment from '@/types/Appointment';
import ErrorDialog from '@/util/ErrorDialog';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faUser, faQuestion, faLink } from '@fortawesome/free-solid-svg-icons';

library.add(faUser, faQuestion, faLink);

@Component
export default class AppointmentsSignUp extends Vue {
	name = '';
	description = '';
	location = '';

	@Prop({ required: true }) queue!: AppointmentsQueue;
	@Prop({ required: true }) time!: Moment;
	@Prop({ required: true }) selectedTimeslot!: number | null;
	@Prop({ required: true }) selectedTime!: Moment | null;

	@Prop({ required: true }) myAppointment!: Appointment | undefined;

	@Watch('myAppointment', { immediate: true })
	myAppointmentUpdated(
		newAppointment: Appointment | undefined,
		oldAppointment: Appointment | undefined
	) {
		if (newAppointment !== undefined) {
			this.name = newAppointment.name || '';
			this.description = newAppointment.description || '';
			this.location = newAppointment.location || '';
			this.$emit(
				'selected',
				newAppointment.timeslot,
				newAppointment.scheduledTime
			);
		}
	}

	get myAppointmentModified() {
		const a = this.myAppointment;
		return (
			a !== undefined &&
			(a.name !== this.name ||
				a.description !== this.description ||
				a.location !== this.location ||
				a.timeslot !== this.selectedTimeslot)
		);
	}

	get canSignUp() {
		return (
			this.$root.$data.loggedIn &&
			this.name !== undefined &&
			this.description !== undefined &&
			this.location !== undefined &&
			this.name.trim() !== '' &&
			this.description.trim() !== '' &&
			this.location.trim() !== ''
		);
	}

	signUp() {
		if (this.queue.confirmSignupMessage !== undefined) {
			return this.$buefy.dialog.confirm({
				title: 'Sign Up',
				message: this.queue.confirmSignupMessage,
				type: 'is-warning',
				hasIcon: true,
				onConfirm: this.signUpRequest,
			});
		}

		this.signUpRequest();
	}

	signUpRequstRunning = false;
	signUpRequest() {
		this.signUpRequstRunning = true;
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/appointments/${this.queue.schedule?.day}/${this.selectedTimeslot}`,
			{
				method: 'POST',
				body: JSON.stringify({
					name: this.name,
					description: this.description,
					location: this.location,
				}),
			}
		).then((res) => {
			this.signUpRequstRunning = false;
			if (res.status !== 201) {
				return ErrorDialog(res);
			}

			this.$buefy.dialog.alert({
				title: 'Appointment Created',
				message: `Your appointment has been created! Make sure to be ready at ${this.selectedTime?.format(
					'LT'
				)}!`,
				type: 'is-success',
				hasIcon: true,
			});
		});
	}

	updateAppointmentRequestRunning = false;
	updateAppointment() {
		this.updateAppointmentRequestRunning = true;
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/appointments/${this.myAppointment?.id}`,
			{
				method: 'PUT',
				body: JSON.stringify({
					name: this.name,
					description: this.description,
					location: this.location,
					timeslot: this.selectedTimeslot,
				}),
			}
		).then((res) => {
			this.updateAppointmentRequestRunning = false;
			if (res.status !== 201) {
				return ErrorDialog(res);
			}

			this.$buefy.dialog.alert({
				title: 'Appointment Updated',
				message: `Your appointment has been updated! Make sure to be ready at ${this.selectedTime?.format(
					'LT'
				)}!`,
				type: 'is-success',
				hasIcon: true,
			});
		});
	}

	cancelAppointmentRequestRunning = false;
	cancelAppointment() {
		this.$buefy.dialog.confirm({
			title: 'Delete Appointment',
			message: `Are you sure you want to cancel your ${this.myAppointment?.scheduledTime.format(
				'LT'
			)} appointment?`,
			type: 'is-danger',
			hasIcon: true,
			onConfirm: () => {
				this.cancelAppointmentRequestRunning = true;
				fetch(
					process.env.BASE_URL +
						`api/queues/${this.queue.id}/appointments/${this.myAppointment?.id}`,
					{
						method: 'DELETE',
					}
				).then((res) => {
					this.cancelAppointmentRequestRunning = false;
					if (res.status !== 204) {
						return ErrorDialog(res);
					}

					this.$buefy.dialog.alert({
						title: 'Appointment Canceled',
						message: `Your appointment has been canceled.`,
						type: 'is-success',
						hasIcon: true,
					});
				});
			},
		});
	}
}
</script>
